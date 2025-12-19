// x/reality/keeper/msg_server_register_node.go
package keeper

import (
	"context"
	"fmt"

	"contactical/x/reality/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (k msgServer) RegisterNode(goCtx context.Context, msg *types.MsgRegisterNode) (*types.MsgRegisterNodeResponse, error) {
    ctx := sdk.UnwrapSDKContext(goCtx)

    // [수정 전] Keeper가 현재 시점 기준으로 챌린지를 재생성 (타이밍 불일치 발생!)
    // blockHash := ctx.BlockHeader().LastBlockId.Hash
    // expectedChallenge := types.GenerateChallengeFromBlockHash(blockHash)

    // [수정 후] 사용자가 제출한 챌린지를 그대로 사용하여 검증
    // (사용자가 "나 이 챌린지 썼어"라고 보낸 값 vs 실제 인증서 안에 박힌 값 비교)
    expectedChallenge := msg.Challenge 

    // 추가 보안: 챌린지가 너무 짧거나 비어있으면 거부
    if len(expectedChallenge) < 16 {
         return nil, status.Error(codes.InvalidArgument, "challenge is too short or empty")
    }

    // 2. Verify attestation certificate
    attestationInfo, err := types.VerifyAttestation(msg.CertChain, expectedChallenge)
    if err != nil {
        return nil, status.Errorf(codes.Unauthenticated, "TEE 검증 실패: %v", err)
    }

    // 3. Validate pub_key
    if len(msg.PubKey) == 0 {
        return nil, status.Error(codes.InvalidArgument, "pub_key cannot be empty")
    }

	// 4. Store node info in blockchain state
	nodeInfo := &types.NodeInfo{
		Creator:          msg.Creator,
		SecurityLevel:    int32(attestationInfo.SecurityLevel),
		DeviceLocked:     attestationInfo.DeviceLocked,
		BootState:        int32(attestationInfo.BootState),
		CreationTime:     attestationInfo.CreationTime,
		AttestationLevel: int32(attestationInfo.AttestationLevel),
		OsVersion:        int32(attestationInfo.OSVersion),
		OsPatchLevel:     int32(attestationInfo.OSPatchLevel),
		RegisteredAt:     ctx.BlockHeight(),
		PubKey:           msg.PubKey, // TEE public key for signature verification
	}

	if err := k.NodeInfo.Set(ctx, msg.Creator, *nodeInfo); err != nil {
		return nil, status.Errorf(codes.Internal, "노드 정보 저장 실패: %v", err)
	}

	// 4. Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"node_registered",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("security_level", fmt.Sprintf("%d", attestationInfo.SecurityLevel)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return &types.MsgRegisterNodeResponse{Success: true}, nil
}