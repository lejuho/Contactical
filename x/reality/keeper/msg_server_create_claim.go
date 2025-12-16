package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	// 1. 컨텍스트 변환 (Go Context -> SDK Context)
	// 블록체인의 상태(State)와 이벤트에 접근하기 위해 필수입니다.
	ctx := sdk.UnwrapSDKContext(goCtx)

	// 2. 주소 유효성 검사 (기존 코드 유지)
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// 3. 비즈니스 로직: 데이터 유효성 검사 (Validation)
	// 빈 데이터가 들어오면 에러를 반환하여 트랜잭션을 거부합니다.
	if msg.SensorHash == "" {
		return nil, fmt.Errorf("sensor hash cannot be empty") // 실제 에러 처리는 types/errors.go에 정의하는 것이 정석이나 지금은 fmt로 처리
	}
	if msg.GnssHash == "" {
		return nil, fmt.Errorf("gnss hash cannot be empty")
	}
	if msg.AnchorSignature == "" {
		return nil, fmt.Errorf("anchor signature cannot be empty")
	}

	// 4. 로깅 (Logging)
	// 노드 운영자가 볼 수 있게 로그를 남깁니다.
	ctx.Logger().Info("Activity Claim Received",
		"Creator", msg.Creator,
		"SensorHash", msg.SensorHash,
		"GnssHash", msg.GnssHash,
	)

	// 5. 이벤트 발생 (Event Emission)
	// 인덱서나 프론트엔드에서 감지할 수 있게 이벤트를 쏩니다.
	ctx.EventManager().EmitEvent(
		sdk.NewEvent("create_claim", // 이벤트 타입 이름
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("sensor_hash", msg.SensorHash),
			sdk.NewAttribute("gnss_hash", msg.GnssHash),
		),
	)

	// 6. 성공 응답
	return &types.MsgCreateClaimResponse{}, nil
}
