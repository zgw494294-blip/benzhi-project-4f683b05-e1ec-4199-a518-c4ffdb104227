package application

import "seed-vault-release/internal/domain"

func ParseRole(value string) (domain.Role, error) {
	role := domain.Role(value)
	switch role {
	case domain.RoleReceiver, domain.RoleTester, domain.RoleReviewer:
		return role, nil
	}
	return "", domain.Invalid("未知角色%s", value)
}

func AllowedActions(c *domain.AccessionCase, role domain.Role) []string {
	actions := []string{}
	if role == domain.RoleReceiver && (c.Status == domain.StatusDraft || c.Status == domain.StatusSampling) {
		actions = append(actions, "修订基础资料", "批量登记批次", "生成或确认抽样")
	}
	if role == domain.RoleTester && (c.Status == domain.StatusTesting || c.Status == domain.StatusRemediation) {
		actions = append(actions, "批量记录检验", "指派整改", "追加证据", "整改与复核关闭")
	}
	if role == domain.RoleReviewer && c.Status == domain.StatusReview {
		actions = append(actions, "审核冻结并签发")
	}
	return actions
}
