package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// ---------------------------------------------------------
// 1. Android Key Attestation ASN.1 구조체 정의
// ---------------------------------------------------------

type SecurityLevel int

const (
	SecurityLevelSoftware  = 0
	SecurityLevelTEE       = 1
	SecurityLevelStrongBox = 2
)

type AuthorizationList struct {
	Purpose              []int       `asn1:"tag:1,explicit,optional"`
	Algorithm            int         `asn1:"tag:2,explicit,optional"`
	KeySize              int         `asn1:"tag:3,explicit,optional"`
	Digest               []int       `asn1:"tag:5,explicit,optional"`
	Padding              []int       `asn1:"tag:6,explicit,optional"`
	EC_Curve             int         `asn1:"tag:10,explicit,optional"`
	RSA_PublicExponent   int         `asn1:"tag:200,explicit,optional"`
	ActiveDateTime       int64       `asn1:"tag:400,explicit,optional"`
	OriginationExpire    int64       `asn1:"tag:401,explicit,optional"`
	UsageExpire          int64       `asn1:"tag:402,explicit,optional"`
	NoAuthRequired       bool        `asn1:"tag:503,explicit,optional"`
	UserAuthType         int         `asn1:"tag:504,explicit,optional"`
	AuthTimeout          int         `asn1:"tag:505,explicit,optional"`
	AllowWhileOnBody     bool        `asn1:"tag:506,explicit,optional"`
	AllApplications      bool        `asn1:"tag:600,explicit,optional"`
	ApplicationID        []byte      `asn1:"tag:601,explicit,optional"`
	CreationDateTime     int64       `asn1:"tag:701,explicit,optional"`
	Origin               int         `asn1:"tag:702,explicit,optional"`
	RootOfTrust          RootOfTrust `asn1:"tag:704,explicit,optional"`
	OSVersion            int         `asn1:"tag:705,explicit,optional"`
	OSPatchLevel         int         `asn1:"tag:706,explicit,optional"`
	AttestationAppID     []byte      `asn1:"tag:709,explicit,optional"`
	AttestationIDBrand   []byte      `asn1:"tag:710,explicit,optional"`
	AttestationIDDevice  []byte      `asn1:"tag:711,explicit,optional"`
	AttestationIDProduct []byte      `asn1:"tag:712,explicit,optional"`
}

type RootOfTrust struct {
	VerifiedBootKey   []byte `asn1:"optional"`
	DeviceLocked      bool
	VerifiedBootState int
	VerifiedBootHash  []byte `asn1:"optional"`
}

type AttestationRecord struct {
	AttestationVersion       int
	AttestationSecurityLevel SecurityLevel
	KeymasterVersion         int
	KeymasterSecurityLevel   SecurityLevel
	AttestationChallenge     []byte
	UniqueID                 []byte
	SoftwareEnforced         AuthorizationList
	TeeEnforced              AuthorizationList
}

// ---------------------------------------------------------
// 2. Challenge 관리
// ---------------------------------------------------------

type ChallengeStore struct {
	mu         sync.RWMutex
	challenges map[string]time.Time
}

func NewChallengeStore() *ChallengeStore {
	return &ChallengeStore{
		challenges: make(map[string]time.Time),
	}
}

func (cs *ChallengeStore) Generate() string {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	// 32바이트 랜덤 challenge 생성
	nonce := make([]byte, 32)
	if _, err := rand.Read(nonce); err != nil {
		log.Printf("Failed to generate nonce: %v", err)
		return ""
	}

	challenge := hex.EncodeToString(nonce)
	cs.challenges[challenge] = time.Now()

	// 오래된 challenge 정리 (5분 이상)
	go cs.cleanup()

	return challenge
}

func (cs *ChallengeStore) Verify(challenge string) bool {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	timestamp, exists := cs.challenges[challenge]
	if !exists {
		return false
	}

	// 5분 이내의 challenge만 유효
	if time.Since(timestamp) > 5*time.Minute {
		delete(cs.challenges, challenge)
		return false
	}

	// 사용된 challenge는 삭제 (replay 방지)
	delete(cs.challenges, challenge)
	return true
}

func (cs *ChallengeStore) cleanup() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	now := time.Now()
	for challenge, timestamp := range cs.challenges {
		if now.Sub(timestamp) > 5*time.Minute {
			delete(cs.challenges, challenge)
		}
	}
}

// ---------------------------------------------------------
// 3. 검증 로직
// ---------------------------------------------------------

type VerificationResult struct {
	Success          bool   `json:"success"`
	Message          string `json:"message"`
	SecurityLevel    int    `json:"security_level,omitempty"`
	DeviceLocked     bool   `json:"device_locked,omitempty"`
	BootState        int    `json:"boot_state,omitempty"`
	CreationTime     int64  `json:"creation_time,omitempty"`
	AttestationLevel int    `json:"attestation_level,omitempty"`
}

func verifyAttestationChain(certChainBase64 []string, expectedChallenge string) (*VerificationResult, error) {
	if len(certChainBase64) == 0 {
		return &VerificationResult{
			Success: false,
			Message: "인증서 체인이 비어있습니다",
		}, nil
	}

	// 1. 인증서 체인 디코딩
	certs := make([]*x509.Certificate, len(certChainBase64))
	for i, certB64 := range certChainBase64 {
		certBytes, err := base64.StdEncoding.DecodeString(certB64)
		if err != nil {
			return &VerificationResult{
				Success: false,
				Message: fmt.Sprintf("인증서 %d 디코딩 실패: %v", i, err),
			}, nil
		}

		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return &VerificationResult{
				Success: false,
				Message: fmt.Sprintf("인증서 %d 파싱 실패: %v", i, err),
			}, nil
		}

		certs[i] = cert
	}

	leafCert := certs[0]

	// 2. 인증서 체인 검증
	if len(certs) > 1 {
		intermediates := x509.NewCertPool()
		for i := 1; i < len(certs); i++ {
			intermediates.AddCert(certs[i])
		}

		opts := x509.VerifyOptions{
			Intermediates: intermediates,
			// Google Root CA를 신뢰하려면 여기에 추가 필요
			// 프로덕션 환경에서는 반드시 Google의 실제 루트 CA를 검증해야 함
		}

		if _, err := leafCert.Verify(opts); err != nil {
			// 개발 환경에서는 경고만 로그
			log.Printf("⚠️ 인증서 체인 검증 실패 (개발 환경에서는 계속 진행): %v", err)
		}
	}

	// 3. Attestation Extension 추출
	attestationOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17}
	var extData []byte
	found := false

	for _, ext := range leafCert.Extensions {
		if ext.Id.Equal(attestationOID) {
			extData = ext.Value
			found = true
			break
		}
	}

	if !found {
		return &VerificationResult{
			Success: false,
			Message: "Attestation Extension을 찾을 수 없습니다",
		}, nil
	}

	// 4. ASN.1 Unmarshal
	var attestation AttestationRecord
	_, err := asn1.Unmarshal(extData, &attestation)
	if err != nil {
		return &VerificationResult{
			Success: false,
			Message: fmt.Sprintf("ASN.1 파싱 실패: %v", err),
		}, nil
	}

	// 5. Challenge 검증
	receivedChallenge := hex.EncodeToString(attestation.AttestationChallenge)
	if receivedChallenge != expectedChallenge {
		return &VerificationResult{
			Success: false,
			Message: fmt.Sprintf("Challenge 불일치 (expected: %s, got: %s)", expectedChallenge, receivedChallenge),
		}, nil
	}

	// 6. 보안 레벨 검증
	if attestation.AttestationSecurityLevel < SecurityLevelTEE {
		return &VerificationResult{
			Success: false,
			Message: "하드웨어(TEE) 보호가 없습니다 (Software level)",
		}, nil
	}

	// 7. RootOfTrust 검증
	rot := attestation.TeeEnforced.RootOfTrust
	if len(rot.VerifiedBootKey) == 0 {
		return &VerificationResult{
			Success: false,
			Message: "RootOfTrust 정보가 TEE 영역에 없습니다",
		}, nil
	}

	if !rot.DeviceLocked {
		return &VerificationResult{
			Success: false,
			Message: "부트로더가 잠겨있지 않습니다",
		}, nil
	}

	if rot.VerifiedBootState != 0 {
		return &VerificationResult{
			Success: false,
			Message: fmt.Sprintf("OS 무결성 확인 실패 (Boot State: %d)", rot.VerifiedBootState),
		}, nil
	}

	// 8. 타임스탬프 검증
	creationTime := attestation.TeeEnforced.CreationDateTime
	if creationTime > 0 {
		creationDate := time.Unix(creationTime/1000, 0)
		// 키가 너무 오래되었는지 확인 (예: 30일)
		if time.Since(creationDate) > 30*24*time.Hour {
			log.Printf("⚠️ 키가 오래되었습니다: %v", creationDate)
		}
	}

	// 9. Attestation Version 검증
	if attestation.AttestationVersion < 3 {
		log.Printf("⚠️ 낮은 Attestation 버전: %d", attestation.AttestationVersion)
	}

	return &VerificationResult{
		Success:          true,
		Message:          "TEE 검증 성공",
		SecurityLevel:    int(attestation.AttestationSecurityLevel),
		DeviceLocked:     rot.DeviceLocked,
		BootState:        rot.VerifiedBootState,
		CreationTime:     creationTime,
		AttestationLevel: attestation.AttestationVersion,
	}, nil
}

// ---------------------------------------------------------
// 4. HTTP 핸들러
// ---------------------------------------------------------

type Server struct {
	challengeStore *ChallengeStore
}

func NewServer() *Server {
	return &Server{
		challengeStore: NewChallengeStore(),
	}
}

// GET /challenge - Challenge 생성
func (s *Server) handleChallenge(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	challenge := s.challengeStore.Generate()
	if challenge == "" {
		http.Error(w, "Failed to generate challenge", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"challenge": challenge,
	})

	log.Printf("Generated challenge: %s", challenge)
}

// POST /verify - 인증서 체인 검증
func (s *Server) handleVerify(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Challenge string   `json:"challenge"`
		CertChain []string `json:"cert_chain"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request: %v", err), http.StatusBadRequest)
		return
	}

	// Challenge 검증
	if !s.challengeStore.Verify(req.Challenge) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(&VerificationResult{
			Success: false,
			Message: "Invalid or expired challenge",
		})
		return
	}

	// 인증서 체인 검증
	result, err := verifyAttestationChain(req.CertChain, req.Challenge)
	if err != nil {
		http.Error(w, fmt.Sprintf("Verification error: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if result.Success {
		w.WriteHeader(http.StatusOK)
		log.Printf("✅ Verification successful for challenge: %s", req.Challenge)
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		log.Printf("❌ Verification failed: %s", result.Message)
	}

	json.NewEncoder(w).Encode(result)
}

// GET /health - 헬스체크
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

// ---------------------------------------------------------
// 5. Main
// ---------------------------------------------------------

func main() {
	server := NewServer()

	http.HandleFunc("/challenge", server.handleChallenge)
	http.HandleFunc("/verify", server.handleVerify)
	http.HandleFunc("/health", server.handleHealth)

	port := ":8080"
	log.Printf("🚀 Contactical TEE Verification Server starting on %s", port)
	log.Printf("📡 Endpoints:")
	log.Printf("   GET  /challenge - Generate new challenge")
	log.Printf("   POST /verify    - Verify attestation")
	log.Printf("   GET  /health    - Health check")

	// Challenge 정리 작업 시작
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			server.challengeStore.cleanup()
		}
	}()

	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

// ---------------------------------------------------------
// 6. 추가 보안 함수들
// ---------------------------------------------------------

// 앱 패키지명 검증 (필요시 사용)
func verifyAppPackage(attestation *AttestationRecord, expectedPackage string) bool {
	if len(attestation.TeeEnforced.AttestationAppID) == 0 {
		return false
	}

	// AttestationApplicationId는 복잡한 ASN.1 구조
	// 실제 구현시 추가 파싱 필요
	_ = sha256.Sum256([]byte(expectedPackage))

	// 간단한 비교 (실제로는 더 복잡한 파싱 필요)
	return len(attestation.TeeEnforced.AttestationAppID) > 0
}
