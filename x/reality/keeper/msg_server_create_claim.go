package keeper

import (
	"context"

	"contactical/x/reality/types"

	errorsmod "cosmossdk.io/errors"
)

func (k msgServer) CreateClaim(ctx context.Context, msg *types.MsgCreateClaim) (*types.MsgCreateClaimResponse, error) {
	if _, err := k.addressCodec.StringToBytes(msg.Creator); err != nil {
		return nil, errorsmod.Wrap(err, "invalid authority address")
	}

	// TODO: Handle the message

	return &types.MsgCreateClaimResponse{}, nil
}
