package domain

type SamplingBand struct {
	MinimumSeedCount int `json:"minimumSeedCount"`
	SampleCount      int `json:"sampleCount"`
}

type SamplingPolicy struct {
	Version         string         `json:"version"`
	Bands           []SamplingBand `json:"bands"`
	MinimumResidual int            `json:"minimumResidual"`
}

func DefaultSamplingPolicy() SamplingPolicy {
	return SamplingPolicy{
		Version: "SVR-2026.1",
		Bands: []SamplingBand{
			{MinimumSeedCount: 200, SampleCount: 50},
			{MinimumSeedCount: 20, SampleCount: 20},
		},
		MinimumResidual: 10,
	}
}

func (p SamplingPolicy) Validate() error {
	if err := RequireText("抽样规则版本", p.Version); err != nil {
		return err
	}
	if len(p.Bands) == 0 {
		return Invalid("抽样规则至少包含一个数量分段")
	}
	previousMinimum := int(^uint(0) >> 1)
	for _, band := range p.Bands {
		if band.MinimumSeedCount < 1 || band.SampleCount < 1 {
			return Invalid("抽样规则分段数值必须大于0")
		}
		if band.MinimumSeedCount >= previousMinimum {
			return Invalid("抽样规则分段必须按最低种子数量降序排列")
		}
		if band.SampleCount > band.MinimumSeedCount {
			return Invalid("计划取样量不得超过分段最低种子数量")
		}
		previousMinimum = band.MinimumSeedCount
	}
	if p.MinimumResidual < 1 {
		return Invalid("最小剩余量必须大于0")
	}
	return nil
}

func (p SamplingPolicy) PlannedCount(seedCount, reservedCount int) (int, error) {
	if err := p.Validate(); err != nil {
		return 0, err
	}
	selected := 0
	for _, band := range p.Bands {
		if seedCount >= band.MinimumSeedCount {
			selected = band.SampleCount
			break
		}
	}
	if selected == 0 {
		return 0, Invalid("种子数量低于抽样规则最低边界")
	}
	available := seedCount - reservedCount
	if available < p.MinimumResidual {
		return 0, Invalid("扣除留样后剩余种子不足")
	}
	if selected > available {
		selected = available
	}
	if selected < p.MinimumResidual {
		return 0, Invalid("可取样数量低于最小检验样本量")
	}
	return selected, nil
}
