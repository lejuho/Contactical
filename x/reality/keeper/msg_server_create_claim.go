package keeper

import (
	"context"
	"crypto/ecdsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"strings"

	"contactical/x/reality/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// [수정] NodeId 대신 Creator를 키로 사용하여 등록된 기기 정보 조회
	// (NodeId 필드는 레거시 혹은 디바이스 고유 ID로 취급)
	nodeInfo, err := k.NodeInfo.Get(ctx, msg.Creator)
	if err != nil {
		return nil, fmt.Errorf("등록되지 않은 노드(기기)입니다. Creator=%s: %w", msg.Creator, err)
	}

	// 파라미터 조회
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, err
	}

	getWeight := func(key string) int64 {
		if val, ok := params.SecurityWeights[key]; ok {
			return int64(val)
		}
		return 0
	}

	// 보안 검증 (개발 모드 or 실제 검증)
	isDevMode := true
	var attResult AttestationResult

	// [ZK-JWT] TrustTier 확인
	isZkVerified := nodeInfo.TrustTier >= 2

	if isDevMode || isZkVerified {
		// ZK 인증된 기기이거나 Dev모드면 TEE 검증 패스 (또는 간소화)
		if isDevMode {
			ctx.Logger().Info("⚠️ [DevMode] Skipping TEE & Signature Verification")
		} else {
			ctx.Logger().Info("🔐 [ZK-Verified] Trusting node based on ZK-JWT tier")
		}
		
		attResult = AttestationResult{
			IsHardwareBacked: true, // ZK 인증도 하드웨어 백킹된 것으로 간주(가정)
			IsStrongBox:      isZkVerified, // ZK 인증은 높은 보안 수준으로 취급
			OSVersion:        140000,
			VerifiedBoot:     "Verified",
		}
	} else {
		// [Legacy] 일반 TEE 기기 검증 로직
		// 1. [재전송 공격 방지] 타임스탬프 검증 (±2분)
		blockTime := ctx.BlockTime().Unix()
		validityWindow := int64(120)

		if msg.Timestamp < blockTime-validityWindow {
			return nil, fmt.Errorf("메시지 만료됨 (재전송 공격 의심): timestamp %d < current %d", msg.Timestamp, blockTime)
		}
		if msg.Timestamp > blockTime+validityWindow {
			return nil, fmt.Errorf("미래의 시간 메시지: timestamp %d > current %d", msg.Timestamp, blockTime)
		}

		// 2. TEE 인증서 검증
		attResult, err = k.ParseAndVerifyTEE(msg.Cert)
		if err != nil {
			return nil, fmt.Errorf("TEE security verification failed: %w", err)
		}

		// 3. [데이터 무결성 검증] 기기 서명 검증 (Payload + Timestamp)
		dataToVerify := fmt.Sprintf("%s%d", msg.Payload, msg.Timestamp)
		
		if !VerifyDeviceSignature(nodeInfo.PubKey, []byte(dataToVerify), msg.DataSignature) {
			return nil, fmt.Errorf("데이터 서명 검증 실패: 기기 키와 일치하지 않음 (위변조 감지)")
		}
	}

	// 신뢰 점수 계산
	var totalScore int64 = 0

	// ZK 인증이면 기본적으로 높은 점수 부여
	if isZkVerified {
		totalScore += 500 // ZK-Bonus (Configurable parameter로 빼는 게 좋음)
	}

	if attResult.IsStrongBox {
		totalScore += getWeight("strongbox")
	} else if attResult.IsHardwareBacked {
		totalScore += getWeight("tee")
	}

	if attResult.VerifiedBoot == "Verified" {
		totalScore += getWeight("boot_lock")
	}

	// [플러그인 시스템 적용] 등록된 검증기(Verifier) 순회
	// 미래의 새로운 보안 모듈(생체인증, ZK, AI분석 등)을 코어 로직 수정 없이 추가 가능
	for _, v := range k.GetVerifiers() {
		// 이 검증기가 처리할 데이터가 있는지 확인
		if v.CanVerify(msg.ExtraAttestation) {
			// 실제 검증 수행 (실패 시 Tx 거부)
			if err := v.Verify(ctx, msg); err != nil {
				return nil, fmt.Errorf("security check failed by plugin '%s': %w", v.Name(), err)
			}
			
			// 검증 성공 시 파라미터 테이블에서 가중치를 찾아 합산
			weight := getWeight(v.Name())
			if weight > 0 {
				totalScore += weight
				ctx.Logger().Info(fmt.Sprintf("🛡️ Plugin Verified: %s (+%d)", v.Name(), weight))
			}
		}
	}

	validNearbyCount := 0
	for _, nodeAddr := range msg.NearbyNodes {
		_, err := k.NodeInfo.Get(ctx, nodeAddr)
		if err == nil {
			validNearbyCount++
		}
	}
	totalScore += int64(validNearbyCount) * getWeight("density_per_node")

	if totalScore > params.MaxTrustScore {
		totalScore = params.MaxTrustScore
	}

	rewardMultiplier := int64(1)

	if totalScore < params.MinScoreThreshold {
		ctx.Logger().Info(fmt.Sprintf("⚠️ Score (%d) below threshold. No reward.", totalScore))
		rewardMultiplier = 0
	} else if isHighPriorityArea(msg.Payload) {
		rewardMultiplier = 2
		ctx.Logger().Info("🚨 [High Priority] Bonus multiplier applied")
	}

	// Claim 저장
	var claim = types.Claim{
		Latitude:         msg.Latitude,
		Longitude:        msg.Longitude,
		Creator:          msg.NodeId, 
		SensorHash:       msg.SensorHash,
		DataSignature:    msg.DataSignature,
		TrustScore:       totalScore,
		RewardMultiplier: rewardMultiplier,
	}
	k.AppendClaim(ctx, claim)

	// 보상 지급
	if rewardMultiplier > 0 {
		rewardAmount := totalScore * rewardMultiplier * params.RewardBaseUnit
		
		if rewardAmount > 0 {
			rewardCoin := sdk.NewCoins(sdk.NewInt64Coin("stake", rewardAmount))

			if err := k.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoin); err != nil {
				return nil, fmt.Errorf("failed to mint coins: %w", err)
			}

			receiver, err := sdk.AccAddressFromBech32(msg.NodeId)
			if err != nil {
				return nil, fmt.Errorf("invalid device address (node_id): %w", err)
			}

			if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, rewardCoin); err != nil {
				return nil, fmt.Errorf("failed to send coins: %w", err)
			}

			ctx.Logger().Info(fmt.Sprintf("💰 Reward Sent to Device: %s (Amount: %s)", msg.NodeId, rewardCoin.String()))
		}
	}

	return &types.MsgCreateClaimResponse{}, nil
}

func isHighPriorityArea(payload string) bool {
	return strings.Contains(payload, "#SOS") ||
		strings.Contains(payload, "#TRUTH") ||
		strings.Contains(payload, "#EMERGENCY")
}

// [신규] 서명 검증 로직 (EC P-256 / RSA 가정)
func VerifyDeviceSignature(pubKeyStr string, data []byte, signatureStr string) bool {
	// 1. PEM 파싱 (안드로이드가 PEM 포맷으로 보냈다고 가정)
	block, _ := pem.Decode([]byte(pubKeyStr))
	var pubKeyBytes []byte
	if block != nil {
		pubKeyBytes = block.Bytes
	} else {
		// PEM이 아니면 Base64 Decode 시도
		var err error
		pubKeyBytes, err = base64.StdEncoding.DecodeString(pubKeyStr)
		if err != nil {
			return false 
		}
	}

	// 2. PublicKey 파싱 (PKIX)
	genericPubKey, err := x509.ParsePKIXPublicKey(pubKeyBytes)
	if err != nil {
		return false
	}

	// 3. 서명 디코딩
	sigBytes, err := base64.StdEncoding.DecodeString(signatureStr)
	if err != nil {
		return false
	}

	// 4. 해시 계산
	h := sha256.New()
	h.Write(data)
	digest := h.Sum(nil)

	// 5. ECDSA 검증
	switch pk := genericPubKey.(type) {
	case *ecdsa.PublicKey:
		return ecdsa.VerifyASN1(pk, digest, sigBytes)
	default:
		return false
	}
}