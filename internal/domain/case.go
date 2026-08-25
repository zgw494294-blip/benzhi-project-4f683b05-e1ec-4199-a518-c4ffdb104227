package domain

import (
	"sort"
	"strings"
	"time"
)

func NewCase(id, code, species, site string, collected, received time.Time, owner string, now time.Time) (*AccessionCase, error) {
	c := &AccessionCase{ID: id, AccessionCode: strings.TrimSpace(code), SpeciesName: strings.TrimSpace(species), CollectionSite: strings.TrimSpace(site), CollectedAt: collected.UTC(), ReceivedAt: received.UTC(), Owner: strings.TrimSpace(owner), Status: StatusDraft, CreatedAt: now.UTC(), UpdatedAt: now.UTC()}
	if err := ValidateNewCase(c); err != nil {
		return nil, err
	}
	return c, nil
}

func (c *AccessionCase) AddLot(lot SeedLot) error {
	return c.AddLots([]SeedLot{lot})
}

func (c *AccessionCase) AddLots(lots []SeedLot) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	if c.Status != StatusDraft && c.Status != StatusSampling {
		return Conflict("当前状态不能登记批次")
	}
	if c.Sampling != nil {
		return Conflict("抽样方案生成后不能继续登记批次")
	}
	if len(lots) == 0 {
		return Invalid("批量登记至少包含一个批次")
	}
	lotCodes := map[string]struct{}{}
	containerCodes := map[string]struct{}{}
	ids := map[string]struct{}{}
	for _, existing := range c.Lots {
		lotCodes[strings.ToLower(strings.TrimSpace(existing.LotCode))] = struct{}{}
		containerCodes[strings.ToLower(strings.TrimSpace(existing.ContainerCode))] = struct{}{}
		ids[existing.ID] = struct{}{}
	}
	prepared := make([]SeedLot, len(lots))
	for i, lot := range lots {
		lot.LotCode = strings.TrimSpace(lot.LotCode)
		lot.ContainerCode = strings.TrimSpace(lot.ContainerCode)
		lot.ArrivalCondition = strings.TrimSpace(lot.ArrivalCondition)
		if err := ValidateLot(lot); err != nil {
			return Invalid("第%d项: %v", i+1, err)
		}
		lotKey := strings.ToLower(lot.LotCode)
		if _, exists := lotCodes[lotKey]; exists {
			return Conflict("第%d项: 批次编号%s重复或已存在", i+1, lot.LotCode)
		}
		containerKey := strings.ToLower(lot.ContainerCode)
		if _, exists := containerCodes[containerKey]; exists {
			return Conflict("第%d项: 容器编号%s重复或已存在", i+1, lot.ContainerCode)
		}
		if _, exists := ids[lot.ID]; exists || strings.TrimSpace(lot.ID) == "" {
			return Conflict("第%d项: 批次标识重复或无效", i+1)
		}
		lotCodes[lotKey] = struct{}{}
		containerCodes[containerKey] = struct{}{}
		ids[lot.ID] = struct{}{}
		lot.CaseID = c.ID
		prepared[i] = lot
	}
	c.Lots = append(c.Lots, prepared...)
	c.Status = StatusSampling
	return nil
}

type CaseRevisionSummary struct {
	ChangedFields []string       `json:"changedFields"`
	Before        map[string]any `json:"before"`
	After         map[string]any `json:"after"`
}

func (c *AccessionCase) ReviseBaseData(species, site string, collected, received time.Time, owner string) (CaseRevisionSummary, error) {
	if err := c.EnsureMutable(); err != nil {
		return CaseRevisionSummary{}, err
	}
	if c.Status != StatusDraft && c.Status != StatusSampling {
		return CaseRevisionSummary{}, Conflict("抽样确认后不能修改案卷基础资料")
	}
	before := baseDataSummary(c)
	next := *c
	next.SpeciesName = strings.TrimSpace(species)
	next.CollectionSite = strings.TrimSpace(site)
	next.CollectedAt = collected.UTC()
	next.ReceivedAt = received.UTC()
	next.Owner = strings.TrimSpace(owner)
	if err := ValidateCaseBasics(&next); err != nil {
		return CaseRevisionSummary{}, err
	}
	after := baseDataSummary(&next)
	changed := make([]string, 0, len(before))
	for field, value := range before {
		if after[field] != value {
			changed = append(changed, field)
		}
	}
	if len(changed) == 0 {
		return CaseRevisionSummary{}, Invalid("没有可保存的资料变更")
	}
	sort.Strings(changed)
	c.SpeciesName, c.CollectionSite = next.SpeciesName, next.CollectionSite
	c.CollectedAt, c.ReceivedAt, c.Owner = next.CollectedAt, next.ReceivedAt, next.Owner
	return CaseRevisionSummary{ChangedFields: changed, Before: before, After: after}, nil
}

func baseDataSummary(c *AccessionCase) map[string]any {
	return map[string]any{
		"speciesName": c.SpeciesName, "collectionSite": c.CollectionSite,
		"collectedAt": c.CollectedAt.UTC().Format(time.RFC3339Nano),
		"receivedAt":  c.ReceivedAt.UTC().Format(time.RFC3339Nano), "owner": c.Owner,
	}
}

func ValidateLot(lot SeedLot) error {
	for _, item := range [][2]string{{"批次编号", lot.LotCode}, {"容器编号", lot.ContainerCode}, {"到库外观", lot.ArrivalCondition}} {
		if err := RequireText(item[0], item[1]); err != nil {
			return err
		}
	}
	if lot.SeedCount < 20 {
		return Invalid("种子数量至少为20")
	}
	if lot.MoisturePercent < 0 || lot.MoisturePercent > 100 {
		return Invalid("含水率必须在0到100之间")
	}
	if lot.ReservedCount < 1 || lot.ReservedCount >= lot.SeedCount {
		return Invalid("留样数量必须大于0且小于种子总数")
	}
	return nil
}

func (c *AccessionCase) ReadyForReview() bool {
	return c.Completeness().Ready
}

func (c *AccessionCase) RefreshStatus() {
	if c.Status == StatusReleased {
		return
	}
	for _, f := range c.Findings {
		if f.Status != FindingClosed {
			c.Status = StatusRemediation
			return
		}
	}
	if c.ReadyForReview() {
		c.Status = StatusReview
		return
	}
	if c.Sampling != nil && c.Sampling.ConfirmedAt != nil {
		c.Status = StatusTesting
		return
	}
	if len(c.Lots) > 0 {
		c.Status = StatusSampling
	} else {
		c.Status = StatusDraft
	}
}
