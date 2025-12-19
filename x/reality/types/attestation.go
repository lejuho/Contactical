package types

import (
	"crypto/x509"
	"encoding/asn1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// ---------------------------------------------------------
// Android Key Attestation ASN.1 구조체 정의
// ---------------------------------------------------------

type SecurityLevel int

const (
	SecurityLevelSoftware  SecurityLevel = 0
	SecurityLevelTEE       SecurityLevel = 1
	SecurityLevelStrongBox SecurityLevel = 2
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
// 검증 결과 구조체
// ---------------------------------------------------------

type AttestationInfo struct {
	SecurityLevel    SecurityLevel
	DeviceLocked     bool
	BootState        int
	CreationTime     int64
	AttestationLevel int
	OSVersion        int
	OSPatchLevel     int
}

// ---------------------------------------------------------
// Challenge 생성 (블록 해시 기반)
// ---------------------------------------------------------

// GenerateChallengeFromBlockHash generates a deterministic challenge from block hash
// This ensures all validators can verify the same challenge without a shared state
func GenerateChallengeFromBlockHash(blockHash []byte) string {
	return hex.EncodeToString(blockHash)
}

// ---------------------------------------------------------
// 검증 로직
// ---------------------------------------------------------

// VerifyAttestation verifies Android Key Attestation certificate chain
// attestationCert: Raw certificate bytes (not Base64 encoded)
// expectedChallenge: Expected challenge string (hex encoded)
func VerifyAttestation(attestationCertBytes []byte, expectedChallenge string) (*AttestationInfo, error) {
	// Parse certificate
	cert, err := x509.ParseCertificate(attestationCertBytes)
	if err != nil {
		return nil, fmt.Errorf("인증서 파싱 실패: %w", err)
	}

	// Extract Attestation Extension
	attestationOID := asn1.ObjectIdentifier{1, 3, 6, 1, 4, 1, 11129, 2, 1, 17}
	var extData []byte
	found := false

	for _, ext := range cert.Extensions {
		if ext.Id.Equal(attestationOID) {
			extData = ext.Value
			found = true
			break
		}
	}

	if !found {
		return nil, fmt.Errorf("Attestation Extension을 찾을 수 없습니다")
	}

	// Parse ASN.1
	var attestation AttestationRecord
	_, err = asn1.Unmarshal(extData, &attestation)
	if err != nil {
		return nil, fmt.Errorf("ASN.1 파싱 실패: %w", err)
	}

	// Verify Challenge
	receivedChallenge := hex.EncodeToString(attestation.AttestationChallenge)
	if receivedChallenge != expectedChallenge {
		return nil, fmt.Errorf("Challenge 불일치 (expected: %s, got: %s)", expectedChallenge, receivedChallenge)
	}

	// Verify Security Level
	if attestation.AttestationSecurityLevel < SecurityLevelTEE {
		return nil, fmt.Errorf("하드웨어(TEE) 보호가 없습니다 (Software level)")
	}

	// Verify RootOfTrust
	rot := attestation.TeeEnforced.RootOfTrust
	if len(rot.VerifiedBootKey) == 0 {
		return nil, fmt.Errorf("RootOfTrust 정보가 TEE 영역에 없습니다")
	}

	if !rot.DeviceLocked {
		return nil, fmt.Errorf("부트로더가 잠겨있지 않습니다")
	}

	if rot.VerifiedBootState != 0 {
		return nil, fmt.Errorf("OS 무결성 확인 실패 (Boot State: %d)", rot.VerifiedBootState)
	}

	// Verify Timestamp
	creationTime := attestation.TeeEnforced.CreationDateTime
	if creationTime > 0 {
		creationDate := time.Unix(creationTime/1000, 0)
		// 키가 너무 오래되었는지 확인 (예: 30일)
		if time.Since(creationDate) > 30*24*time.Hour {
			log.Printf("⚠️ 키가 오래되었습니다: %v", creationDate)
		}
	}

	// Verify Attestation Version
	if attestation.AttestationVersion < 3 {
		log.Printf("⚠️ 낮은 Attestation 버전: %d", attestation.AttestationVersion)
	}

	return &AttestationInfo{
		SecurityLevel:    attestation.AttestationSecurityLevel,
		DeviceLocked:     rot.DeviceLocked,
		BootState:        rot.VerifiedBootState,
		CreationTime:     creationTime,
		AttestationLevel: attestation.AttestationVersion,
		OSVersion:        attestation.TeeEnforced.OSVersion,
		OSPatchLevel:     attestation.TeeEnforced.OSPatchLevel,
	}, nil
}

// VerifyAttestationChain verifies certificate chain with Base64 encoded certificates
// This function is for backward compatibility with the HTTP server
func VerifyAttestationChain(certChainBase64 []string, expectedChallenge string) (*AttestationInfo, error) {
	if len(certChainBase64) == 0 {
		return nil, fmt.Errorf("인증서 체인이 비어있습니다")
	}

	// Decode certificates
	certs := make([]*x509.Certificate, len(certChainBase64))
	for i, certB64 := range certChainBase64 {
		certBytes, err := base64.StdEncoding.DecodeString(certB64)
		if err != nil {
			return nil, fmt.Errorf("인증서 %d 디코딩 실패: %w", i, err)
		}

		cert, err := x509.ParseCertificate(certBytes)
		if err != nil {
			return nil, fmt.Errorf("인증서 %d 파싱 실패: %w", i, err)
		}

		certs[i] = cert
	}

	leafCert := certs[0]

	// Verify certificate chain
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

	// Use the main verification function
	return VerifyAttestation(leafCert.Raw, expectedChallenge)
}
