package domain

import (
	"strings"
	"time"
)

func (c *AccessionCase) GenerateSamplingPlan(id string) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusSampling || len(c.Lots) == 0 {
		return Conflict("仅待抽样且已有批次的案卷可以生成方案")
	}
	if c.Sampling != nil && c.Sampling.ConfirmedAt != nil {
		return Conflict("已确认的抽样方案禁止重新生成或覆盖")
	}
	policy := DefaultSamplingPolicy()
	items := make([]SamplingItem, 0, len(c.Lots))
	for _, lot := range c.Lots {
		planned, err := policy.PlannedCount(lot.SeedCount, lot.ReservedCount)
		if err != nil {
			return Invalid("批次%s无法制定抽样计划: %v", lot.LotCode, err)
		}
		items = append(items, SamplingItem{LotID: lot.ID, LotCode: lot.LotCode, PlannedCount: planned, ReservedCount: lot.ReservedCount})
	}
	c.Sampling = &SamplingPlan{ID: id, CaseID: c.ID, RuleVersion: policy.Version, Items: items}
	return nil
}

func (c *AccessionCase) ConfirmSampling(actual map[string]int, actor string, now time.Time) error {
	return c.ConfirmSamplingWithDeviations(actual, nil, actor, now)
}

func (c *AccessionCase) ConfirmSamplingWithDeviations(actual map[string]int, deviations map[string]string, actor string, now time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusSampling || c.Sampling == nil {
		return Conflict("尚未生成可确认的抽样方案")
	}
	if c.Sampling.ConfirmedAt != nil {
		return Conflict("抽样方案已确认")
	}
	if err := RequireText("确认人", actor); err != nil {
		return err
	}
	planned := make(map[string]struct{}, len(c.Sampling.Items))
	for _, item := range c.Sampling.Items {
		planned[item.LotID] = struct{}{}
	}
	for lotID := range actual {
		if _, ok := planned[lotID]; !ok {
			return Invalid("实际取样量包含未知批次%s", lotID)
		}
	}
	for lotID := range deviations {
		if _, ok := planned[lotID]; !ok {
			return Invalid("偏差说明包含未知批次%s", lotID)
		}
	}
	type confirmed struct {
		actual, available int
		reason            string
	}
	values := make([]confirmed, len(c.Sampling.Items))
	for i := range c.Sampling.Items {
		item := c.Sampling.Items[i]
		amount, ok := actual[item.LotID]
		if !ok {
			return Invalid("缺少批次%s的实际取样量", item.LotCode)
		}
		lot, _ := c.Lot(item.LotID)
		if amount < item.PlannedCount {
			return Invalid("批次%s实际取样量低于计划", item.LotCode)
		}
		if amount+item.ReservedCount > lot.SeedCount {
			return Invalid("批次%s取样后无法满足留样边界", item.LotCode)
		}
		reason := strings.TrimSpace(deviations[item.LotID])
		if amount != item.PlannedCount && reason == "" {
			return Invalid("批次%s实际取样量偏离计划时必须填写偏差原因", item.LotCode)
		}
		values[i] = confirmed{actual: amount, available: lot.SeedCount - amount - item.ReservedCount, reason: reason}
	}
	for i := range c.Sampling.Items {
		c.Sampling.Items[i].ActualCount = values[i].actual
		c.Sampling.Items[i].AvailableCount = values[i].available
		c.Sampling.Items[i].DeviationReason = values[i].reason
	}
	t := now.UTC()
	c.Sampling.ConfirmedBy = actor
	c.Sampling.ConfirmedAt = &t
	c.Status = StatusTesting
	return nil
}
