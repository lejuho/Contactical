package keeper_test

import (
	"testing"

	"contactical/x/reality/types"

	"github.com/stretchr/testify/require"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params:     types.DefaultParams(),
		ClaimList:  []types.Claim{{Id: 0}, {Id: 1}},
		ClaimCount: 2,
	}
	f := initFixture(t)
	err := f.keeper.InitGenesis(f.ctx, genesisState)
	require.NoError(t, err)
	got, err := f.keeper.ExportGenesis(f.ctx)
	require.NoError(t, err)
	require.NotNil(t, got)

	require.EqualExportedValues(t, genesisState.Params, got.Params)
	require.EqualExportedValues(t, genesisState.ClaimList, got.ClaimList)
	require.Equal(t, genesisState.ClaimCount, got.ClaimCount)

}
