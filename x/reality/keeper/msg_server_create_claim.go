package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 1. Node 등록 여부 확인 (변수 미사용 에러 해결)
	// nodeInfo 자체는 지금 안 쓰므로 '_'로 받아서 에러 여부만 체크
	_, err := k.NodeInfo.Get(ctx, msg.Creator)
	if err != nil {
		// 등록되지 않은 노드일 경우의 로직 (로그만 남기고 통과 등)
		ctx.Logger().Info("Unregistered node submitting claim", "creator", msg.Creator)
	}

	// 2. 데이터 서명 검증 (리턴값 개수 & 대소문자 해결)
	// msg.Payload (대문자 P), 리턴값 2개(isValid, err) 처리
	isValid, err := types.VerifyDataSignature(msg.Payload, msg.DataSignature, msg.Cert)
	if err != nil {
		return nil, fmt.Errorf("signature verification error: %v", err)
	}
	if !isValid {
		return nil, fmt.Errorf("invalid signature")
	}

	// 3. 중복 검사
	isDuplicate, err := k.IsSensorHashDuplicated(ctx, msg.SensorHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplication: %w", err)
	}
	if isDuplicate {
		return nil, fmt.Errorf("duplicate sensor hash: claim already exists")
	}

	// 4. 저장 (Claim 생성)
	var claim = types.Claim{
		Creator:         msg.Creator,
		SensorHash:      msg.SensorHash,
		GnssHash:        msg.GnssHash,
		AnchorSignature: msg.AnchorSignature,
		DataSignature:   msg.DataSignature,
		// TrustLevel 등의 필드는 필요 시 로직 추가
	}

	id, err := k.AppendClaim(ctx, claim)
	if err != nil {
		return nil, err
	}

	// 5. 이벤트 발생
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("create_claim",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("id", fmt.Sprintf("%d", id)),
		),
	)

	return &types.MsgCreateClaimResponse{}, nil
}