package keeper

import (
	"fmt"

	"cosmossdk.io/collections"
	"cosmossdk.io/core/address"
	corestore "cosmossdk.io/core/store"
	"github.com/cosmos/cosmos-sdk/codec"

	"contactical/x/reality/types"
)

type Keeper struct {
	storeService corestore.KVStoreService
	cdc          codec.Codec
	addressCodec address.Codec

	// Address capable of executing a MsgUpdateParams message.
	// Typically, this should be the x/gov module account.
	authority []byte

	Schema collections.Schema
	Params collections.Item[types.Params]

	bankKeeper    types.BankKeeper
	stakingKeeper types.StakingKeeper
	ClaimSeq      collections.Sequence
	Claim         collections.Map[uint64, types.Claim]
	NodeInfo      collections.Map[string, types.NodeInfo]
}

func NewKeeper(
	storeService corestore.KVStoreService,
	cdc codec.Codec,
	addressCodec address.Codec,
	authority []byte,

	bankKeeper types.BankKeeper,
	stakingKeeper types.StakingKeeper,
) Keeper {
	if _, err := addressCodec.BytesToString(authority); err != nil {
		panic(fmt.Sprintf("invalid authority address %s: %s", authority, err))
	}

	sb := collections.NewSchemaBuilder(storeService)

	k := Keeper{
		storeService: storeService,
		cdc:          cdc,
		addressCodec: addressCodec,
		authority:    authority,

		bankKeeper:    bankKeeper,
		stakingKeeper: stakingKeeper,
		Params:        collections.NewItem(sb, types.ParamsKey, "params", codec.CollValue[types.Params](cdc)),
		Claim:         collections.NewMap(sb, types.ClaimKey, "claim", collections.Uint64Key, codec.CollValue[types.Claim](cdc)),
		ClaimSeq:      collections.NewSequence(sb, types.ClaimCountKey, "claimSequence"),
		NodeInfo:      collections.NewMap(sb, types.NodeInfoKey, "nodeInfo", collections.StringKey, codec.CollValue[types.NodeInfo](cdc)),
	}
	schema, err := sb.Build()
	if err != nil {
		panic(err)
	}
	k.Schema = schema

	return k
}

// GetAuthority returns the module's authority.
func (k Keeper) GetAuthority() []byte {
	return k.authority
}
