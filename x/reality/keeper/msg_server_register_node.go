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
		"zk_mode", len(msg.Nullifier) > 0,
	)

	nodeInfo := &types.NodeInfo{
		Creator:      msg.Creator,
		RegisteredAt: ctx.BlockHeight(),
		PubKey:       msg.PubKey,
	}

	// [ZK-JWT Mode] Nullifier가 존재하면 ZK 인증으로 간주
	if len(msg.Nullifier) > 0 {
		// 1. Nullifier 중복 체크 (Double Registration / Double Spending 방지)
		has, err := k.Nullifiers.Has(ctx, msg.Nullifier)
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to check nullifier")
		}
		if has {
			return nil, status.Error(codes.AlreadyExists, "nullifier already used: node already registered")
		}

		// 2. Nullifier 저장 (KeySet 사용)
		if err := k.Nullifiers.Set(ctx, msg.Nullifier); err != nil {
			return nil, status.Error(codes.Internal, "failed to store nullifier")
		}

		// 3. NodeInfo 설정
		nodeInfo.Nullifier = msg.Nullifier
		nodeInfo.TrustTier = 2 // 2 = ZK-Verified (Trustworthy)

		// TODO: 온체인 ZK Proof 검증을 원하면 이곳에 Verifier 로직 추가
		// 현재는 Proxy가 검증했다고 가정하고 Pass

		ctx.Logger().Info("🔐 ZK-JWT Node Registered", "creator", msg.Creator, "nullifier", msg.Nullifier)

	} else {
		// [Legacy/TEE Mode] 기존 Android Attestation 로직
		expectedChallenge := msg.Challenge
		if len(expectedChallenge) == 0 {
			// ZK 모드도 아니고 TEE 모드도 아니면 에러
			return nil, status.Error(codes.InvalidArgument, "challenge or nullifier required")
		}

		// TEE 인증서 검증 (개발 모드: 실패해도 막지 않음)
		attestationInfo, err := types.VerifyAttestation(msg.CertChain, expectedChallenge)
		if err != nil {
			ctx.Logger().Error("⚠️ TEE verification failed (dev mode, ignoring)", "err", err)
			attestationInfo = &types.AttestationInfo{
				SecurityLevel:    1,
				DeviceLocked:     true,
				BootState:        1,
				CreationTime:     ctx.BlockTime().Unix(),
				AttestationLevel: 1,
				OSVersion:        1,
				OSPatchLevel:     1,
			}
		}

		nodeInfo.SecurityLevel = int32(attestationInfo.SecurityLevel)
		nodeInfo.DeviceLocked = attestationInfo.DeviceLocked
		nodeInfo.BootState = int32(attestationInfo.BootState)
		nodeInfo.CreationTime = attestationInfo.CreationTime
		nodeInfo.AttestationLevel = int32(attestationInfo.AttestationLevel)
		nodeInfo.OsVersion = int32(attestationInfo.OSVersion)
		nodeInfo.OsPatchLevel = int32(attestationInfo.OSPatchLevel)
		nodeInfo.TrustTier = 1 // 1 = Basic/Legacy
	}

	// 4. 최종 NodeInfo 저장
	if err := k.NodeInfo.Set(ctx, msg.Creator, *nodeInfo); err != nil {
		return nil, status.Errorf(codes.Internal, "노드 정보 저장 실패: %v", err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"node_registered",
			sdk.NewAttribute("creator", msg.Creator),
			sdk.NewAttribute("trust_tier", fmt.Sprintf("%d", nodeInfo.TrustTier)),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	return &types.MsgRegisterNodeResponse{Success: true}, nil
}
