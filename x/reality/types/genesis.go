package types

import "fmt"

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:    DefaultParams(),
		ClaimList: []Claim{}}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	claimIdMap := make(map[uint64]bool)
	claimCount := gs.GetClaimCount()
	for _, elem := range gs.ClaimList {
		if _, ok := claimIdMap[elem.Id]; ok {
			return fmt.Errorf("duplicated id for claim")
		}
		if elem.Id >= claimCount {
			return fmt.Errorf("claim id should be lower or equal than the last id")
		}
		claimIdMap[elem.Id] = true
	}

	return gs.Params.Validate()
}
