package keeper

import (
	"contactical/x/reality/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Verifier defines the interface for pluggable security verifiers.
// This allows future security modules (e.g. Biometrics, ZK-Proofs) to be added
// without modifying the core logic.
type Verifier interface {
	// Name returns the unique identifier for this verifier.
	// This name is used to lookup the weight in Params.SecurityWeights.
	Name() string

	// CanVerify checks if the verifier should run based on the input data.
	CanVerify(extra map[string]string) bool

	// Verify performs the cryptographic or logic validation.
	// If it returns an error, the claim creation is rejected (strict security).
	Verify(ctx sdk.Context, msg *types.MsgCreateClaim) error
}
