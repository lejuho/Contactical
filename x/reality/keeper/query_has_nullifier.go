package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k queryServer) HasNullifier(goCtx context.Context, req *types.QueryHasNullifierRequest) (*types.QueryHasNullifierResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	has, err := k.k.Nullifiers.Has(ctx, req.Nullifier)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("failed to check nullifier: %v", err))
	}

	return &types.QueryHasNullifierResponse{HasNullifier: has}, nil
}
