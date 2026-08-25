package application

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

type CaseDetail struct {
	Case                *domain.AccessionCase     `json:"case"`
	Timeline            []audit.Event             `json:"timeline"`
	AuditValid          bool                      `json:"auditValid"`
	AuditMessage        string                    `json:"auditMessage"`
	Completeness        domain.CompletenessReport `json:"completeness"`
	AllowedActions      map[domain.Role][]string  `json:"allowedActions"`
	PossibleTransitions []domain.TransitionRule   `json:"possibleTransitions"`
	Assessments         []domain.TestAssessment   `json:"assessments"`
	TestMatrix          []domain.TestMatrixCell   `json:"testMatrix"`
}

type CertificateQuery struct {
	SerialNumber   uint64
	AccessionCode  string
	ManifestDigest string
}

type VerificationCheck struct {
	Code    string `json:"code"`
	Label   string `json:"label"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

type CertificateVerification struct {
	Valid           bool                       `json:"valid"`
	Message         string                     `json:"message"`
	Certificate     *domain.ReleaseCertificate `json:"certificate,omitempty"`
	AccessionCode   string                     `json:"accessionCode,omitempty"`
	Checks          []VerificationCheck        `json:"checks"`
	ManifestSummary json.RawMessage            `json:"manifestSummary,omitempty"`
}

func (s *Service) GetCase(id string) (*CaseDetail, error) {
	c, err := s.store.GetCase(id)
	if err != nil {
		return nil, mapStoreError(err)
	}
	for i := range c.Findings {
		c.Findings[i].Timeliness = c.Findings[i].Timing(s.now())
	}
	events, err := s.store.Timeline(id)
	if err != nil {
		return nil, err
	}
	detail := &CaseDetail{Case: c, Timeline: events, AuditValid: true, AuditMessage: "审计摘要链有效"}
	detail.Completeness = c.Completeness()
	detail.AllowedActions = map[domain.Role][]string{
		domain.RoleReceiver: AllowedActions(c, domain.RoleReceiver),
		domain.RoleTester:   AllowedActions(c, domain.RoleTester),
		domain.RoleReviewer: AllowedActions(c, domain.RoleReviewer),
	}
	detail.PossibleTransitions = c.PossibleTransitions()
	detail.Assessments, err = c.Assessments()
	if err != nil {
		return nil, err
	}
	detail.TestMatrix, err = c.TestMatrix()
	if err != nil {
		return nil, err
	}
	if err := audit.Validate(events); err != nil {
		detail.AuditValid, detail.AuditMessage = false, err.Error()
	}
	return detail, nil
}

func (s *Service) ListCases(status domain.Status) ([]domain.AccessionCase, error) {
	return s.ListCasesByFindingTiming(status, "")
}

func (s *Service) ListCasesByFindingTiming(status domain.Status, timing string) ([]domain.AccessionCase, error) {
	if err := domain.ValidateStatusFilter(status); err != nil {
		return nil, err
	}
	if timing != "" && timing != "未到期" && timing != "临期" && timing != "逾期" && timing != "未设置" {
		return nil, domain.Invalid("未知发现项时效状态%s", timing)
	}
	items, cached, epoch := s.cachedCaseList(status)
	if !cached {
		loaded, err := s.store.ListCases(status)
		if err != nil {
			return nil, err
		}
		items = loaded
		s.rememberCaseList(status, epoch, items)
	}
	result := make([]domain.AccessionCase, 0, len(items))
	for _, item := range items {
		matched := timing == ""
		for i := range item.Findings {
			item.Findings[i].Timeliness = item.Findings[i].Timing(s.now())
			if item.Findings[i].Timeliness == timing {
				matched = true
			}
		}
		if matched {
			result = append(result, item)
		}
	}
	return result, nil
}

func (s *Service) VerifyCertificate(serial uint64) (*CertificateVerification, error) {
	return s.VerifyCertificateQuery(CertificateQuery{SerialNumber: serial})
}

func (s *Service) VerifyCertificateQuery(query CertificateQuery) (*CertificateVerification, error) {
	count := 0
	if query.SerialNumber != 0 {
		count++
	}
	if strings.TrimSpace(query.AccessionCode) != "" {
		count++
	}
	if strings.TrimSpace(query.ManifestDigest) != "" {
		count++
	}
	if count == 0 {
		return nil, domain.Invalid("必须提供凭据序号、案卷编号或完整清单摘要之一")
	}
	if count != 1 {
		return nil, domain.Invalid("核验条件存在歧义，每次只能提供一种精确条件")
	}

	var cert *domain.ReleaseCertificate
	var c *domain.AccessionCase
	storedCertificate := true
	var err error
	if query.SerialNumber != 0 {
		cert, err = s.store.CertificateBySerial(query.SerialNumber)
		if err == nil {
			c, err = s.store.GetCase(cert.CaseID)
		}
	} else if code := strings.TrimSpace(query.AccessionCode); code != "" {
		c, err = s.store.CaseByAccession(code)
		if err == nil && c.Certificate == nil {
			return nil, domain.NotFound("该案卷没有放行凭据")
		}
		if err == nil {
			cert, err = s.store.CertificateBySerial(c.Certificate.SerialNumber)
			if err == repository.ErrNotFound {
				storedCertificate, cert, err = false, c.Certificate, nil
			}
		}
	} else {
		digest := strings.ToLower(strings.TrimSpace(query.ManifestDigest))
		decoded, decodeErr := hex.DecodeString(digest)
		if decodeErr != nil || len(decoded) != 32 || len(digest) != 64 {
			return nil, domain.Invalid("清单摘要必须是64位十六进制完整摘要")
		}
		cert, err = s.store.CertificateByDigest(digest)
		if err == nil {
			c, err = s.store.GetCase(cert.CaseID)
		}
	}
	if err != nil {
		return nil, mapStoreError(err)
	}
	result := &CertificateVerification{Certificate: cert, AccessionCode: c.AccessionCode, Checks: []VerificationCheck{}}
	add := func(code, label string, passed bool, message string) {
		result.Checks = append(result.Checks, VerificationCheck{Code: code, Label: label, Passed: passed, Message: message})
	}
	recordOK := storedCertificate && cert.ID != "" && cert.SerialNumber > 0 && cert.CaseID != ""
	add("credential_record", "只追加凭据记录", recordOK, passMessage(recordOK, "凭据记录存在且字段完整", "凭据记录不存在或字段不完整"))
	caseMatch := c.Certificate != nil && sameCertificate(cert, c.Certificate)
	add("case_credential", "案卷内凭据", caseMatch, passMessage(caseMatch, "案卷内凭据与只追加记录一致", "案卷内凭据序号、标识、摘要或签发信息不一致"))
	released := c.Status == domain.StatusReleased
	add("released_status", "案卷放行状态", released, passMessage(released, "案卷状态为已放行", "案卷不是已放行状态"))
	revisionMatch := cert.CaseRevision == c.Revision
	add("case_revision", "签发案卷版本", revisionMatch, passMessage(revisionMatch, fmt.Sprintf("签发版本与当前v%d一致", c.Revision), fmt.Sprintf("凭据版本v%d与案卷v%d不一致", cert.CaseRevision, c.Revision)))

	storedManifest, manifestErr := s.store.ManifestBySerial(cert.SerialNumber)
	manifestRecord := manifestErr == nil && c.FrozenManifest != nil && sameManifest(storedManifest, c.FrozenManifest)
	add("manifest_record", "只追加冻结清单", manifestRecord, passMessage(manifestRecord, "冻结清单记录与案卷一致", "冻结清单记录缺失、损坏或与案卷不一致"))
	canonical, digest, buildErr := domain.BuildManifest(c)
	digestMatch := buildErr == nil && c.FrozenManifest != nil && canonical == c.FrozenManifest.CanonicalJSON && digest == c.FrozenManifest.Digest && digest == cert.ManifestDigest
	add("canonical_digest", "规范清单摘要", digestMatch, passMessage(digestMatch, "重新构建的规范清单及摘要一致", "规范清单内容或摘要与冻结记录不一致"))

	events, timelineErr := s.store.Timeline(c.ID)
	chainValid := timelineErr == nil && audit.Validate(events) == nil
	add("audit_chain", "审计摘要链", chainValid, passMessage(chainValid, "审计事件摘要链连续有效", "审计事件摘要链损坏"))
	freezeFound, issueFound := audit.ValidateReleaseEvents(events, cert)
	add("freeze_event", "冻结审计事件", freezeFound, passMessage(freezeFound, "冻结事件与签发版本和摘要一致", "缺少匹配的冻结审计事件"))
	add("issue_event", "签发审计事件", issueFound, passMessage(issueFound, "签发事件位于冻结事件之后且信息一致", "缺少匹配的签发审计事件或事件次序不正确"))
	if manifestRecord {
		result.ManifestSummary, _ = domain.ManifestBusinessSummary(storedManifest.CanonicalJSON)
	}
	result.Valid = true
	failures := []string{}
	for _, check := range result.Checks {
		if !check.Passed {
			result.Valid = false
			failures = append(failures, check.Label+"："+check.Message)
		}
	}
	if result.Valid {
		result.Message = "放行凭据有效，冻结清单与审计链逐项核验通过"
	} else {
		result.Message = "核验失败：" + strings.Join(failures, "；")
	}
	return result, nil
}

func passMessage(ok bool, pass, fail string) string {
	if ok {
		return pass
	}
	return fail
}

func sameCertificate(a, b *domain.ReleaseCertificate) bool {
	return a != nil && b != nil && a.ID == b.ID && a.CaseID == b.CaseID && a.SerialNumber == b.SerialNumber && a.ManifestDigest == b.ManifestDigest && a.CaseRevision == b.CaseRevision && a.ApprovedBy == b.ApprovedBy && a.IssuedAt.Equal(b.IssuedAt)
}

func sameManifest(a, b *domain.FrozenManifest) bool {
	return a != nil && b != nil && a.CanonicalJSON == b.CanonicalJSON && a.Digest == b.Digest && a.FrozenBy == b.FrozenBy && a.FrozenAt.Equal(b.FrozenAt)
}
