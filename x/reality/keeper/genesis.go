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

	return genesis, nil
}
