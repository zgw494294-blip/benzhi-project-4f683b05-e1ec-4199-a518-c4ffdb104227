package domain

import (
	"strings"
	"time"
)

const findingDueSoonWindow = 72 * time.Hour

func (c *AccessionCase) AssignFinding(id, assignee string, dueAt time.Time) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	f, err := c.Finding(id)
	if err != nil {
		return err
	}
	if f.Status == FindingClosed {
		return Conflict("已关闭发现项不得重新指派")
	}
	if err := RequireText("整改负责人", assignee); err != nil {
		return err
	}
	if dueAt.IsZero() || !dueAt.After(f.CreatedAt) {
		return Invalid("整改截止时间必须晚于发现时间")
	}
	t := dueAt.UTC()
	f.Assignee, f.DueAt = strings.TrimSpace(assignee), &t
	return nil
}

func (c *AccessionCase) AddFindingEvidence(id string, evidence FindingEvidence) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	f, err := c.Finding(id)
	if err != nil {
		return err
	}
	if f.Status != FindingOpen {
		return Conflict("仅整改处理中的发现项可以追加证据")
	}
	if err := RequireText("证据提交人", evidence.SubmittedBy); err != nil {
		return err
	}
	if err := RequireText("证据说明", evidence.Description); err != nil {
		return err
	}
	if err := RequireText("证据内容摘要", evidence.ContentDigest); err != nil {
		return err
	}
	for _, existing := range f.EvidenceLedger {
		if existing.ID == evidence.ID {
			return Conflict("证据记录标识重复")
		}
	}
	evidence.Description = strings.TrimSpace(evidence.Description)
	evidence.SubmittedBy = strings.TrimSpace(evidence.SubmittedBy)
	evidence.SubmittedAt = evidence.SubmittedAt.UTC()
	f.EvidenceLedger = append(f.EvidenceLedger, evidence)
	return nil
}

// RemediateFinding 保留领域层旧调用的兼容语义；公开应用流程使用
// SubmitFindingRemediation，并要求证据先通过只追加台账写入。
func (c *AccessionCase) RemediateFinding(id, evidence, actor, retestID string) error {
	f, err := c.Finding(id)
	if err != nil {
		return err
	}
	originalEvidenceCount := len(f.EvidenceLedger)
	if originalEvidenceCount == 0 && strings.TrimSpace(evidence) != "" {
		retest, testErr := c.Test(retestID)
		if testErr != nil {
			return Invalid("必须关联有效复验记录")
		}
		f.EvidenceLedger = append(f.EvidenceLedger, FindingEvidence{ID: "legacy-" + retestID, SubmittedBy: actor, Description: strings.TrimSpace(evidence), ContentDigest: DigestText(evidence), SubmittedAt: retest.PerformedAt})
	}
	if err := c.SubmitFindingRemediation(id, actor, retestID); err != nil {
		f.EvidenceLedger = f.EvidenceLedger[:originalEvidenceCount]
		return err
	}
	return nil
}

func (c *AccessionCase) SubmitFindingRemediation(id, actor, retestID string) error {
	if err := c.EnsureMutable(); err != nil {
		return err
	}
	f, err := c.Finding(id)
	if err != nil {
		return err
	}
	if f.Status != FindingOpen {
		return Conflict("发现项当前不能提交整改")
	}
	if len(f.EvidenceLedger) == 0 {
		return Invalid("提交复验前必须至少追加一条处置证据")
	}
	if err := RequireText("整改人", actor); err != nil {
		return err
	}
	retest, err := c.Test(retestID)
	if err != nil {
		return Invalid("必须关联有效复验记录")
	}
	source, err := c.Test(f.SourceTestID)
	if err != nil {
		return Invalid("发现项来源检验不存在")
	}
	if !retest.IsRetest || retest.LotID != f.LotID || retest.TestType != source.TestType {
		return Invalid("复验记录与发现项批次或检验类型不匹配")
	}
	if retest.Result != ResultPass {
		return Invalid("复验未通过，不能提交关闭")
	}
	f.RemediationEvidence = DigestEvidenceLedger(f.EvidenceLedger)
	f.RemediatedBy, f.RetestID, f.Status = strings.TrimSpace(actor), retestID, FindingRemediated
	c.Status = StatusRemediation
	return nil
}

func (c *AccessionCase) CloseFinding(id, actor string, role Role, now time.Time) error {
	if err := ValidateRole(role, RoleTester); err != nil {
		return err
	}
	f, err := c.Finding(id)
	if err != nil {
		return err
	}
	if f.Status != FindingRemediated {
		return Conflict("发现项尚未完成整改与复验")
	}
	retest, retestErr := c.Test(f.RetestID)
	source, sourceErr := c.Test(f.SourceTestID)
	if retestErr != nil || sourceErr != nil || !retest.IsRetest || retest.Result != ResultPass || retest.LotID != f.LotID || retest.TestType != source.TestType {
		return Invalid("关闭前必须保有通过的同批次同类型复验")
	}
	responsible := f.Assignee
	if responsible == "" {
		responsible = f.RemediatedBy
	}
	if strings.EqualFold(strings.TrimSpace(actor), strings.TrimSpace(responsible)) {
		return Forbidden("整改负责人不能复核关闭自己的发现项")
	}
	if err := RequireText("关闭人", actor); err != nil {
		return err
	}
	t := now.UTC()
	f.Status, f.ClosedBy, f.ClosedAt = FindingClosed, strings.TrimSpace(actor), &t
	c.RefreshStatus()
	return nil
}

func (f Finding) Timing(now time.Time) string {
	if f.Status == FindingClosed {
		return "已关闭"
	}
	if f.DueAt == nil {
		return "未设置"
	}
	remaining := f.DueAt.Sub(now.UTC())
	if remaining < 0 {
		return "逾期"
	}
	if remaining <= findingDueSoonWindow {
		return "临期"
	}
	return "未到期"
}

func DigestEvidenceLedger(items []FindingEvidence) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, item.ID+"\x00"+item.SubmittedBy+"\x00"+item.Description+"\x00"+item.ContentDigest+"\x00"+item.SubmittedAt.UTC().Format(time.RFC3339Nano))
	}
	return DigestText(strings.Join(parts, "\x01"))
}
