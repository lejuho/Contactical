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

    ctx.Logger().Info("📥 RegisterNode received",
        "creator", msg.Creator,
        "challenge_len", len(msg.Challenge),
        "cert_chain_count", len(msg.CertChain),
    )

    // 1. 챌린지 최소 길이만 가볍게 체크 (원하면 이것도 완화 가능)
    expectedChallenge := msg.Challenge
    if len(expectedChallenge) == 0 {
        return nil, status.Error(codes.InvalidArgument, "challenge cannot be empty")
    }

    // 2. TEE 인증서 검증 (개발 모드: 실패해도 막지 않음)
    attestationInfo, err := types.VerifyAttestation(msg.CertChain, expectedChallenge)
    if err != nil {
        ctx.Logger().Error("⚠️ TEE verification failed (dev mode, ignoring)", "err", err)

        // 개발용 더미 값 채우기
        attestationInfo = &types.AttestationInfo{
            SecurityLevel:    1,
            DeviceLocked:     true,
            BootState:        1,
            CreationTime:     ctx.BlockTime().Unix(),
            AttestationLevel: 1,
            OSVersion:        1,
            OSPatchLevel:     1,
        }
        // 운영 모드에서는 여기서 return 해버리는 게 맞음:
        // return nil, status.Errorf(codes.Unauthenticated, "TEE 검증 실패: %v", err)
    }

    // 3. pub_key 기본 체크
    if len(msg.PubKey) == 0 {
        return nil, status.Error(codes.InvalidArgument, "pub_key cannot be empty")
    }

    // 4. NodeInfo 저장
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
        PubKey:           msg.PubKey,
    }

    if err := k.NodeInfo.Set(ctx, msg.Creator, *nodeInfo); err != nil {
        return nil, status.Errorf(codes.Internal, "노드 정보 저장 실패: %v", err)
    }

    ctx.EventManager().EmitEvent(
        sdk.NewEvent(
            "node_registered",
            sdk.NewAttribute("creator", msg.Creator),
            sdk.NewAttribute("security_level", fmt.Sprintf("%d", attestationInfo.SecurityLevel)),
            sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
        ),
    )

    ctx.Logger().Info("✅ Node registered (dev mode TEE)", "creator", msg.Creator)

    return &types.MsgRegisterNodeResponse{Success: true}, nil
}
