package types

// NewParams creates a new Params instance with default values.
func NewParams() Params {
	return Params{
		RewardBaseUnit:      1000,
		BonusStrongbox:      50,
		BonusTee:            30,
		BonusBootLock:       10,
		BonusDensityPerNode: 20,
		MaxTrustScore:       100,
	}
}

// DefaultParams returns a default set of parameters.
func DefaultParams() Params {
	return NewParams()
}

// Validate validates the set of params.
func (p Params) Validate() error {
	// 여기에 reward_base_unit이 0보다 큰지 등의 체크 로직을 넣을 수 있습니다.
	return nil
}
