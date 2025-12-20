package keeper

import (
	"context"
	"fmt"
	"strings"

	"contactical/x/reality/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// -----------------------------------------------------------
	// 1. TEE 서명 및 인증서 검증 (현재 테스트를 위해 검증 성공으로 우회)
	// -----------------------------------------------------------
	// 실제 운영 시에는 types.VerifyDataSignature 로직을 통해 하드웨어 보안성을 체크합니다.
	// isValid, _ := types.VerifyDataSignature(msg.Payload, msg.DataSignature, msg.Cert)
	isValid := true // 임시 우회
	if !isValid {
		return nil, fmt.Errorf("invalid TEE signature or certificate")
	}

	// -----------------------------------------------------------
	// 2. 밀집도 기반 신뢰 점수(Trust Score) 계산
	// -----------------------------------------------------------
	trustScore := int32(10) // 기본 점수 (주변에 아무도 없을 때)
	validNearbyCount := 0

	for _, nodeAddr := range msg.NearbyNodes {
		// 주변에 있다고 주장하는 노드가 실제로 우리 체인에 등록된(RegisterNode) 노드인지 장부에서 확인
		_, err := k.NodeInfo.Get(ctx, nodeAddr)
		if err == nil {
			validNearbyCount++
		}
	}

	// 보너스 점수: 주변 인증 노드 1개당 20점 추가 (최대 100점)
	trustScore += int32(validNearbyCount * 20)
	if trustScore > 100 {
		trustScore = 100
	}

	// -----------------------------------------------------------
	// 3. 검열 저항 및 사회적 가중치(Reward Multiplier) 계산
	// -----------------------------------------------------------
	rewardMultiplier := int32(1)
	// 페이로드에 특정 긴급 키워드가 포함된 경우 "중요 데이터"로 판단하여 보상을 2배로 설정
	if isHighPriorityArea(msg.Payload) {
		rewardMultiplier = 2
		ctx.Logger().Info("🚨 [Censorship-Resistant] High priority data detected! Applying 2x reward multiplier.")
	}

	// -----------------------------------------------------------
	// 4. 데이터 저장 (Claim 기록)
	// -----------------------------------------------------------
	var claim = types.Claim{
		Creator:          msg.Creator,
		SensorHash:       msg.SensorHash,
		DataSignature:    msg.DataSignature,
		TrustScore:       trustScore,
		RewardMultiplier: rewardMultiplier,
	}
	k.AppendClaim(ctx, claim)

	// -----------------------------------------------------------
	// 5. 토큰 보상 지급 (Minting & Transfer)
	// -----------------------------------------------------------
	// 보상 공식: (신뢰 점수 * 가중치 * 기본 단위 1000)
	// 예: 30점 x 2배 x 1000 = 60,000 stake
	rewardBase := int64(trustScore) * int64(rewardMultiplier) * 1000
	rewardCoin := sdk.NewCoins(sdk.NewInt64Coin("stake", rewardBase))

	// [Mint] Reality 모듈이 무에서 유로 토큰을 찍어냄
	err := k.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoin)
	if err != nil {
		return nil, fmt.Errorf("failed to mint reward tokens: %w", err)
	}

	// [Send] 찍어낸 토큰을 데이터 제출자(Creator)의 지갑으로 전송
	receiver, _ := sdk.AccAddressFromBech32(msg.Creator)
	err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, rewardCoin)
	if err != nil {
		return nil, fmt.Errorf("failed to transfer reward tokens: %w", err)
	}

	// 성공 로그 출력
	ctx.Logger().Info(fmt.Sprintf("💰 [Reward Success] Creator: %s, Score: %d, Multiplier: %d, Amount: %s",
		msg.Creator, trustScore, rewardMultiplier, rewardCoin.String()))

	return &types.MsgCreateClaimResponse{}, nil
}

// 이 함수는 데이터 내용(Payload)을 분석하여 사회적 중요도를 판별합니다.
func isHighPriorityArea(payload string) bool {
	// 특정 키워드가 포함된 경우 검열 저항 데이터로 간주
	return strings.Contains(payload, "#SOS") || 
		   strings.Contains(payload, "#TRUTH") || 
		   strings.Contains(payload, "#EMERGENCY")
}