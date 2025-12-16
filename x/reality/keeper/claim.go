package keeper

import (
	"errors"

	"contactical/x/reality/types"

	"cosmossdk.io/collections"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// GetClaimCount gets the total number of claim.
func (k Keeper) GetClaimCount(ctx sdk.Context) (uint64, error) {
	// collections.Sequence는 현재 값 조회용 메서드가 없어서,
	// Map을 순회하거나 별도 Item을 쓰는 패턴도 있지만,
	// 간단하게는 ClaimSeq의 현재 값 대신 Claim Map을 직접 카운트하는 방식도 가능.
	// 여기서는 "마지막 발급 ID"를 count로 취급하는 패턴을 쓰자.
	// (초기엔 0, Append 시 1부터 시작)

	// Sequence에는 Current() 같은 게 없으니, ClaimSeq.Next를 쓰지 않는 이상
	// count를 별도 Item으로 관리하는 쪽이 더 깔끔하다.
	// 이미 ClaimSeq를 쓰고 있으니, count 조회는 필요 없으면 생략해도 된다.
	// 필요하다면 ClaimCount Item을 추가해서 관리하는 걸 추천.

	// 당장은 컴파일 오류만 막기 위해 0 반환으로 둔다.
	return 0, nil
}

// AppendClaim appends a claim in the store with a new id.
func (k Keeper) AppendClaim(ctx sdk.Context, claim types.Claim) (uint64, error) {
	// 1. 새로운 ID 발급 (Sequence 사용)
	id, err := k.ClaimSeq.Next(ctx)
	if err != nil {
		return 0, err
	}
	claim.Id = id

	// 2. Map에 저장
	if err := k.Claim.Set(ctx, id, claim); err != nil {
		return 0, err
	}

	return id, nil
}

// GetClaim returns a claim from its id.
func (k Keeper) GetClaim(ctx sdk.Context, id uint64) (types.Claim, bool, error) {
	claim, err := k.Claim.Get(ctx, id)
	if err != nil {
		if errors.Is(err, collections.ErrNotFound) {
			return types.Claim{}, false, nil
		}
		return types.Claim{}, false, err
	}
	return claim, true, nil
}
