package keeper

import (
	"context"
	"contactical/x/contactical/types"
    // "cosmossdk.io/collections" // Ignite 버전에 따라 다름
)

// IsSensorHashDuplicated: 저장된 모든 Claim을 뒤져서 SensorHash가 같은 게 있는지 확인
func (k Keeper) IsSensorHashDuplicated(ctx context.Context, sensorHash string) (bool, error) {
	var found bool = false

	// Ignite가 생성한 k.Claim (Map 혹은 Collection)을 순회
    // 주의: 데이터가 많아지면 이 방식(Walk)은 느려집니다. 
    // 나중에는 SensorHash를 Key로 하는 별도의 Index Map을 만들어야 합니다.
	err := k.Claim.Walk(ctx, nil, func(key uint64, val types.Claim) (stop bool, err error) {
		if val.SensorHash == sensorHash {
			found = true
			return true, nil // Stop iteration
		}
		return false, nil
	})

	if err != nil {
		return false, err
	}

	return found, nil
}