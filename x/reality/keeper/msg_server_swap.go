package keeper

import (
    "context"
    "fmt"

    "contactical/x/reality/types"

    sdk "github.com/cosmos/cosmos-sdk/types"
    authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
) 

func (k msgServer) Swap(goCtx context.Context, msg *types.MsgSwap) (*types.MsgSwapResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // 1. 입력값 파싱
    amountInCoin, err := sdk.ParseCoinNormalized(msg.AmountIn)
    if err != nil {
        return nil, fmt.Errorf("invalid amount_in: %w", err)
    }
    creatorAddr, err := sdk.AccAddressFromBech32(msg.Creator)
    if err != nil {
        return nil, fmt.Errorf("invalid creator address: %w", err)
    }

    // 2. [핵심] 유동성 풀(금고) 잔고 확인
    moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
    reserveIn := k.bankKeeper.GetBalance(ctx, moduleAddr, amountInCoin.Denom).Amount.ToLegacyDec()
    reserveOut := k.bankKeeper.GetBalance(ctx, moduleAddr, msg.TargetDenom).Amount.ToLegacyDec()

    if reserveIn.IsZero() || reserveOut.IsZero() {
        return nil, fmt.Errorf("환전소 금고가 비어있습니다 (유동성 부족)")
    }

    // 3. Constant Product 공식 적용: dy = (y * dx) / (x + dx)
    dx := amountInCoin.Amount.ToLegacyDec()
    dy := reserveOut.Mul(dx).Quo(reserveIn.Add(dx))

    rewardCoinOut := sdk.NewCoin(msg.TargetDenom, dy.TruncateInt())

    // 4. 실제 돈 이동 (사용자 -> 모듈 / 모듈 -> 사용자)
    err = k.bankKeeper.SendCoins(ctx, creatorAddr, moduleAddr, sdk.NewCoins(amountInCoin))
    if err != nil { 
        return nil, fmt.Errorf("failed to send coins to module: %w", err)
    }

    err = k.bankKeeper.SendCoins(ctx, moduleAddr, creatorAddr, sdk.NewCoins(rewardCoinOut))
    if err != nil { 
        return nil, fmt.Errorf("failed to send coins to user: %w", err)
    }

    ctx.Logger().Info(fmt.Sprintf("💱 [DEX Swap] %s → %s", amountInCoin.String(), rewardCoinOut.String()))

    return &types.MsgSwapResponse{AmountOut: rewardCoinOut.String()}, nil
}
