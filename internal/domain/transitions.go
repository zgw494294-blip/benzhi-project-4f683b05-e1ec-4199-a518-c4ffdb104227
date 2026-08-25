package domain

type TransitionRule struct {
	From            Status `json:"from"`
	To              Status `json:"to"`
	Action          string `json:"action"`
	ResponsibleRole Role   `json:"responsibleRole"`
	Condition       string `json:"condition"`
}

var workflowTransitions = []TransitionRule{
	{From: StatusDraft, To: StatusSampling, Action: "登记首个种子批次", ResponsibleRole: RoleReceiver, Condition: "批次关键字段和留样边界有效"},
	{From: StatusSampling, To: StatusTesting, Action: "确认抽样方案", ResponsibleRole: RoleReceiver, Condition: "所有批次实际取样量达到计划且保留留样"},
	{From: StatusTesting, To: StatusRemediation, Action: "记录异常检验", ResponsibleRole: RoleTester, Condition: "发芽率未达下限或污染率超过上限"},
	{From: StatusTesting, To: StatusReview, Action: "完成全部检验", ResponsibleRole: RoleTester, Condition: "每批两类初检齐全且没有未关闭发现项"},
	{From: StatusRemediation, To: StatusReview, Action: "关闭全部发现项", ResponsibleRole: RoleTester, Condition: "处置证据和通过复验齐全，并由不同检验员关闭"},
	{From: StatusReview, To: StatusReleased, Action: "审核冻结并签发", ResponsibleRole: RoleReviewer, Condition: "审核员与案卷负责人职责分离，完整性检查通过"},
}

func WorkflowTransitions() []TransitionRule {
	return append([]TransitionRule(nil), workflowTransitions...)
}

func (c *AccessionCase) PossibleTransitions() []TransitionRule {
	result := []TransitionRule{}
	for _, rule := range workflowTransitions {
		if rule.From == c.Status {
			result = append(result, rule)
		}
	}
	return result
}

func IsKnownStatus(status Status) bool {
	switch status {
	case StatusDraft, StatusSampling, StatusTesting, StatusRemediation, StatusReview, StatusReleased:
		return true
	default:
		return false
	}
}

func ValidateStatusFilter(status Status) error {
	if status == "" || IsKnownStatus(status) {
		return nil
	}
	return Invalid("未知案卷状态%s", status)
}
