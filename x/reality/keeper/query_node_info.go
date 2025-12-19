package keeper

import (
	"context"

	"contactical/x/reality/types"

	"cosmossdk.io/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetNodeInfo queries node information by creator address
func (qs queryServer) GetNodeInfo(goCtx context.Context, req *types.QueryGetNodeInfoRequest) (*types.QueryGetNodeInfoResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	if req.Creator == "" {
		return nil, status.Error(codes.InvalidArgument, "creator address cannot be empty")
	}

	// Retrieve node info from state
	nodeInfo, err := qs.k.NodeInfo.Get(goCtx, req.Creator)
	if err != nil {
		return nil, errors.Wrapf(err, "node info not found for creator: %s", req.Creator)
	}

	return &types.QueryGetNodeInfoResponse{
		NodeInfo: nodeInfo,
	}, nil
}
