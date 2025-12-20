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

    // [신규] 현재 체인의 보상 정책(Params)을 가져옵니다.
    params, err := k.GetParams(ctx)
    if err != nil {
        return nil, err
    }

    // [1] 보안 검증 단계
    isDevMode := true 
    var attResult AttestationResult

    if isDevMode {
        attResult = AttestationResult{
            IsHardwareBacked: true,
            IsStrongBox:      true,
            OSVersion:        140000,
            VerifiedBoot:     "Verified",
        }
    } else {
        attResult, err = k.ParseAndVerifyTEE(msg.Cert)
        if err != nil {
            return nil, fmt.Errorf("보안 검증 실패: %w", err)
        }
    }

    // [2] 통합 신뢰 점수 계산 (파라미터 기반)
    var totalScore int32 = 10 // 기본 점수

    // 2-1. 보안 등급 가산점 (파라미터 사용)
    if attResult.IsStrongBox {
        totalScore += params.BonusStrongbox
    } else if attResult.IsHardwareBacked {
        totalScore += params.BonusTee
    }

    // 2-2. 부트 상태 가산점 (파라미터 사용)
    if attResult.VerifiedBoot == "Verified" {
        totalScore += params.BonusBootLock
    }

    // 2-3. 밀집도 가산점 (파라미터 사용)
    validNearbyCount := 0
    for _, nodeAddr := range msg.NearbyNodes {
        _, err := k.NodeInfo.Get(ctx, nodeAddr)
        if err == nil {
            validNearbyCount++
        }
    }
    totalScore += int32(validNearbyCount) * params.BonusDensityPerNode

    // 최대 점수 제한 (파라미터 사용)
    if totalScore > params.MaxTrustScore {
        totalScore = params.MaxTrustScore
    }

    // [3] 보상 가중치 계산
    rewardMultiplier := int32(1)
    if isHighPriorityArea(msg.Payload) {
        rewardMultiplier = 2
        ctx.Logger().Info("🚨 [Censorship-Resistant] High priority data detected!")
    }

    // [4] 데이터 저장
    var claim = types.Claim{
        Creator:          msg.Creator,
        SensorHash:       msg.SensorHash,
        DataSignature:    msg.DataSignature,
        TrustScore:       totalScore,
        RewardMultiplier: rewardMultiplier,
    }
    k.AppendClaim(ctx, claim)

    // [5] 토큰 보상 지급 (파라미터 기반)
    rewardBase := int64(totalScore) * int64(rewardMultiplier) * params.RewardBaseUnit
    rewardCoin := sdk.NewCoins(sdk.NewInt64Coin("stake", rewardBase))

    // MintCoins
    err = k.bankKeeper.MintCoins(ctx, types.ModuleName, rewardCoin)
    if err != nil {
        return nil, fmt.Errorf("failed to mint reward tokens: %w", err)
    }

    // Send
    receiver, err := sdk.AccAddressFromBech32(msg.Creator)
    if err != nil {
        return nil, fmt.Errorf("invalid creator address: %w", err)
    }

    err = k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, receiver, rewardCoin)
    if err != nil {
        return nil, fmt.Errorf("failed to transfer reward tokens: %w", err)
    }

    ctx.Logger().Info(fmt.Sprintf("💰 [Reward Success] Score: %d, Amount: %s", totalScore, rewardCoin.String()))

    return &types.MsgCreateClaimResponse{}, nil
}

// 이 함수는 데이터 내용(Payload)을 분석하여 사회적 중요도를 판별합니다.
func isHighPriorityArea(payload string) bool {
    return strings.Contains(payload, "#SOS") || 
           strings.Contains(payload, "#TRUTH") || 
           strings.Contains(payload, "#EMERGENCY")
}
