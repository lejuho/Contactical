package keeper

import (
	"context"

	"contactical/x/reality/types"
)

// InitGenesis initializes the module's state from a provided genesis state.
func (k Keeper) InitGenesis(ctx context.Context, genState types.GenesisState) error {
	for _, elem := range genState.ClaimList {
		if err := k.Claim.Set(ctx, elem.Id, elem); err != nil {
			return err
		}
	}

	if err := k.ClaimSeq.Set(ctx, genState.ClaimCount); err != nil {
		return err
	}

	// Set all the nodeInfo
	for _, elem := range genState.NodeList {
		if err := k.NodeInfo.Set(ctx, elem.Creator, elem); err != nil {
			return err
		}
	}

	// Set all the nullifier
	for _, elem := range genState.NullifierList {
		if err := k.Nullifiers.Set(ctx, elem); err != nil {
			return err
		}
	}

	return k.Params.Set(ctx, genState.Params)
}

// ExportGenesis returns the module's exported genesis.
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	var err error

	genesis := types.DefaultGenesis()
	genesis.Params, err = k.Params.Get(ctx)
	if err != nil {
		return nil, err
	}
	err = k.Claim.Walk(ctx, nil, func(key uint64, elem types.Claim) (bool, error) {
		genesis.ClaimList = append(genesis.ClaimList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	genesis.ClaimCount, err = k.ClaimSeq.Peek(ctx)
	if err != nil {
		return nil, err
	}

	// Get all nodeInfo
	err = k.NodeInfo.Walk(ctx, nil, func(key string, elem types.NodeInfo) (bool, error) {
		genesis.NodeList = append(genesis.NodeList, elem)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	// Get all nullifier
	err = k.Nullifiers.Walk(ctx, nil, func(key string) (bool, error) {
		genesis.NullifierList = append(genesis.NullifierList, key)
		return false, nil
	})
	if err != nil {
		return nil, err
	}

	return genesis, nil
}
