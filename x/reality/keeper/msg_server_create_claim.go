package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func (k msgServer) CreateClaim(goCtx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	if msg.SensorHash == "" {
		return nil, fmt.Errorf("sensor hash cannot be empty")
	}
	if msg.GnssHash == "" {
		return nil, fmt.Errorf("gnss hash cannot be empty")
	}
	// AnchorSignature 필수 체크 삭제

	var verificationStatus string
	if msg.AnchorSignature == "" {
		verificationStatus = "GPS_ONLY"
		ctx.Logger().Info("Claim accepted without Anchor (Low Trust)")
	} else {
		if len(msg.AnchorSignature) < 10 {
			return nil, fmt.Errorf("invalid anchor signature format")
		}
		verificationStatus = "ANCHOR_VERIFIED"
		ctx.Logger().Info("Claim accepted with Valid Anchor (High Trust)")
	}

	isDuplicate, err := k.IsSensorHashDuplicated(ctx, msg.SensorHash)
	if err != nil {
		return nil, fmt.Errorf("failed to check duplication: %w", err)
	}
	if isDuplicate {
		return nil, fmt.Errorf("duplicate sensor hash: claim already exists")
	}

	var claim = types.Claim{
		Creator:         msg.Creator,
		SensorHash:      msg.SensorHash,
		GnssHash:        msg.GnssHash,
		AnchorSignature: msg.AnchorSignature,
		TrustLevel:      verificationStatus,
	}

	id, err := k.AppendClaim(ctx, claim)
	if err != nil {
		return nil, err
	}
	claim.Id = id

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("create_claim",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("sensor_hash", msg.SensorHash),
			sdk.NewAttribute("trust_level", verificationStatus),
			sdk.NewAttribute("id", fmt.Sprintf("%d", claim.Id)),
		),
	)

	return &types.MsgCreateClaimResponse{}, nil
}
