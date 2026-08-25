package domain

import (
	"strings"
	"time"
)

func RequireText(name, value string) error {
	if strings.TrimSpace(value) == "" {
		return Invalid("%s不能为空", name)
	}
	return nil
}

func ValidateRole(actual Role, allowed ...Role) error {
	for _, role := range allowed {
		if actual == role {
			return nil
		}
	}
	return Forbidden("角色%s无权执行此操作", actual)
}

func ValidateNewCase(c *AccessionCase) error {
	if err := RequireText("案卷编号", c.AccessionCode); err != nil {
		return err
	}
	return ValidateCaseBasics(c)
}

func ValidateCaseBasics(c *AccessionCase) error {
	checks := [][2]string{{"物种名称", c.SpeciesName}, {"采集地点", c.CollectionSite}, {"负责人", c.Owner}}
	for _, check := range checks {
		if err := RequireText(check[0], check[1]); err != nil {
			return err
		}
	}
	if c.CollectedAt.IsZero() || c.ReceivedAt.IsZero() {
		return Invalid("采集时间和到库时间不能为空")
	}
	if c.ReceivedAt.Before(c.CollectedAt) {
		return Invalid("到库时间不得早于采集时间")
	}
	if c.ReceivedAt.After(time.Now().Add(24 * time.Hour)) {
		return Invalid("到库时间不合理")
	}
	return nil
}

func (c *AccessionCase) EnsureMutable() error {
	if c.Status == StatusReleased || c.FrozenManifest != nil {
		return Conflict("案卷已冻结，禁止修改")
	}
	return nil
}

func (c *AccessionCase) Lot(id string) (*SeedLot, error) {
	for i := range c.Lots {
		if c.Lots[i].ID == id {
			return &c.Lots[i], nil
		}
	}
	return nil, NotFound("种子批次不存在")
}

func (c *AccessionCase) Test(id string) (*ViabilityTest, error) {
	for i := range c.Tests {
		if c.Tests[i].ID == id {
			return &c.Tests[i], nil
		}
	}
	return nil, NotFound("检验记录不存在")
}

func (c *AccessionCase) Finding(id string) (*Finding, error) {
	for i := range c.Findings {
		if c.Findings[i].ID == id {
			return &c.Findings[i], nil
		}
	}
	return nil, NotFound("发现项不存在")
}
