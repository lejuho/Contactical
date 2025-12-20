// x/reality/keeper/query_node_info.go
package keeper

import (
    "context"

    "contactical/x/reality/types"

    "cosmossdk.io/errors"
    sdk "github.com/cosmos/cosmos-sdk/types"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

// 단일 노드 조회
func (qs queryServer) GetNodeInfo(goCtx context.Context, req *types.QueryGetNodeInfoRequest) (*types.QueryGetNodeInfoResponse, error) {
    if req == nil {
        return nil, status.Error(codes.InvalidArgument, "invalid request")
    }

    if req.Creator == "" {
        return nil, status.Error(codes.InvalidArgument, "creator address cannot be empty")
    }

    nodeInfo, err := qs.k.NodeInfo.Get(goCtx, req.Creator)
    if err != nil {
        return nil, errors.Wrapf(err, "node info not found for creator: %s", req.Creator)
    }

    return &types.QueryGetNodeInfoResponse{
        NodeInfo: nodeInfo,
    }, nil
}

// 전체 노드 조회
func (qs queryServer) AllNodeInfo(goCtx context.Context, req *types.QueryAllNodeInfoRequest) (*types.QueryAllNodeInfoResponse, error) {
    if req == nil {
        return nil, status.Error(codes.InvalidArgument, "invalid request")
    }

    ctx := sdk.UnwrapSDKContext(goCtx)

    var nodeInfos []*types.NodeInfo

    err := qs.k.NodeInfo.Walk(ctx, nil, func(key string, value types.NodeInfo) (bool, error) {
        v := value
        nodeInfos = append(nodeInfos, &v)
        return false, nil
    })
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }

    return &types.QueryAllNodeInfoResponse{
        NodeInfo: nodeInfos,
    }, nil
}
