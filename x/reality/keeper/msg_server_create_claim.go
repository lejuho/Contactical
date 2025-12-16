package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	// 1. 컨텍스트 변환
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 2. 주소 유효성 검사
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// 3. 데이터 유효성 검사
	if msg.SensorHash == "" {
		return nil, fmt.Errorf("sensor hash cannot be empty")
	}
	if msg.GnssHash == "" {
		return nil, fmt.Errorf("gnss hash cannot be empty")
	}
	if msg.AnchorSignature == "" {
		return nil, fmt.Errorf("anchor signature cannot be empty")
	}

	// 4. 로깅
	ctx.Logger().Info("Activity Claim Received",
		"Creator", msg.Creator,
		"SensorHash", msg.SensorHash,
	)

	// 5. [수정됨] 데이터 저장 (저장 로직)
	// Creator 필드는 scaffold 할 때 자동으로 안 생겼을 수 있으므로,
	// 만약 'unknown field Creator' 에러가 계속 나면 이 줄을 지워야 합니다.
	// 우선은 시도해 봅시다.
	var claim = types.Claim{
		Creator:         msg.Creator,
		SensorHash:      msg.SensorHash,
		GnssHash:        msg.GnssHash,
		AnchorSignature: msg.AnchorSignature,
	}

	// [핵심 수정] k.AppendClaim -> k.Keeper.AppendClaim
	// msgServer 구조체 내부에 있는 실제 Keeper 객체를 통해 호출해야 합니다.
	id, err := k.AppendClaim(ctx, claim)
	if err != nil {
		return nil, err
	}
	claim.Id = id

	// 6. 이벤트 발생
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("create_claim",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("sensor_hash", msg.SensorHash),
			sdk.NewAttribute("id", fmt.Sprintf("%d", claim.Id)), // 저장된 ID도 이벤트에 포함
		),
	)

	return &types.MsgCreateClaimResponse{}, nil
}
