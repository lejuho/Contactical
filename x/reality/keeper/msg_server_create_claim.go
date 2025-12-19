package keeper

import (
	"context"
	"fmt"
	"time"

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

	// ============================================================
	// TEE Signature Verification (V6.0 Hardware-Native Verification)
	// ============================================================

	var verificationStatus string

	// Check if node is registered and verify TEE signature
	nodeInfo, err := k.NodeInfo.Get(ctx, msg.Creator)
	if err != nil {
		// Node not registered - allow with low trust level
		ctx.Logger().Info("Node not registered, accepting with LOW_TRUST", "creator", msg.Creator)
		verificationStatus = "UNREGISTERED_NODE"
	} else {
		// Node is registered - verify TEE signature
		if len(msg.DataSignature) == 0 {
			return nil, fmt.Errorf("data_signature required for registered nodes")
		}

		// Verify timestamp is recent (within 5 minutes)
		now := time.Now().Unix()
		if msg.Timestamp == 0 || now-msg.Timestamp > 300 || msg.Timestamp-now > 60 {
			return nil, fmt.Errorf("invalid timestamp: must be within 5 minutes")
		}

		// Verify signature using registered TEE public key
		if err := types.VerifyDataSignatureAuto(
			nodeInfo.PubKey,
			msg.DataSignature,
			msg.SensorHash,
			msg.GnssHash,
			msg.Timestamp,
		); err != nil {
			return nil, fmt.Errorf("TEE signature verification failed: %w", err)
		}

		ctx.Logger().Info("TEE signature verified successfully", "creator", msg.Creator)

		// Determine trust level based on anchor + TEE
		if msg.AnchorSignature == "" {
			verificationStatus = "TEE_VERIFIED"
		} else {
			verificationStatus = "TEE_AND_ANCHOR_VERIFIED"
		}
	}

	// Legacy anchor-only verification for unregistered nodes
	if verificationStatus == "UNREGISTERED_NODE" {
		if msg.AnchorSignature == "" {
			verificationStatus = "GPS_ONLY"
			ctx.Logger().Info("Claim accepted without Anchor (Low Trust)")
		} else {
			if len(msg.AnchorSignature) < 10 {
				return nil, fmt.Errorf("invalid anchor signature format")
			}
			verificationStatus = "ANCHOR_ONLY"
			ctx.Logger().Info("Claim accepted with Anchor only (Medium Trust)")
		}
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
