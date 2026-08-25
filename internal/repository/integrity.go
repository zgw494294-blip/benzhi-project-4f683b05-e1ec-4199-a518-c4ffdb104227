package repository

import (
	"encoding/json"
	"fmt"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
)

type IntegrityReport struct {
	Valid           bool     `json:"valid"`
	CaseCount       int      `json:"caseCount"`
	ReleasedCount   int      `json:"releasedCount"`
	AuditEventCount int      `json:"auditEventCount"`
	Errors          []string `json:"errors"`
}

func cloneIntegrityReport(report IntegrityReport) IntegrityReport {
	report.Errors = append([]string(nil), report.Errors...)
	return report
}

func (s *Store) cachedIntegrityReport() (IntegrityReport, bool) {
	s.integrityMu.Lock()
	defer s.integrityMu.Unlock()
	if s.integrityCached == nil {
		return IntegrityReport{}, false
	}
	return cloneIntegrityReport(*s.integrityCached), true
}

func (s *Store) rememberIntegrityReport(report IntegrityReport) {
	s.integrityMu.Lock()
	defer s.integrityMu.Unlock()
	cached := cloneIntegrityReport(report)
	s.integrityCached = &cached
}

func (s *Store) CheckIntegrity() (IntegrityReport, error) {
	if cached, ok := s.cachedIntegrityReport(); ok {
		return cached, nil
	}
	report := IntegrityReport{Valid: true, Errors: []string{}}
	err := s.db.View(func(tx *bolt.Tx) error {
		if err := validateSchema(tx); err != nil {
			return err
		}
		return tx.Bucket(bucketCases).ForEach(func(key, value []byte) error {
			report.CaseCount++
			var c domain.AccessionCase
			if err := json.Unmarshal(value, &c); err != nil {
				report.Errors = append(report.Errors, fmt.Sprintf("案卷%s无法解码", key))
				return nil
			}
			if c.ID != string(key) {
				report.Errors = append(report.Errors, "案卷键与记录ID不一致")
			}
			indexValue := tx.Bucket(bucketStatus).Get(statusKey(string(c.Status), c.ID))
			if string(indexValue) != c.ID {
				report.Errors = append(report.Errors, "案卷"+c.ID+"状态索引缺失")
			}
			events, err := loadTimeline(tx, c.ID)
			if err != nil {
				report.Errors = append(report.Errors, "案卷"+c.ID+"时间线无法读取")
				return nil
			}
			report.AuditEventCount += len(events)
			if err := audit.Validate(events); err != nil {
				report.Errors = append(report.Errors, "案卷"+c.ID+"审计链无效: "+err.Error())
			}
			if c.Status == domain.StatusReleased {
				report.ReleasedCount++
				if c.Certificate == nil || c.FrozenManifest == nil {
					report.Errors = append(report.Errors, "已放行案卷"+c.ID+"缺少凭据或清单")
					return nil
				}
				stored := tx.Bucket(bucketCertificates).Get(certificateKey(c.Certificate.SerialNumber))
				if stored == nil {
					report.Errors = append(report.Errors, "已放行案卷"+c.ID+"缺少只追加凭据记录")
				}
				valid, message := domain.VerifyCertificate(&c)
				if !valid {
					report.Errors = append(report.Errors, "已放行案卷"+c.ID+"凭据无效: "+message)
				}
			}
			return nil
		})
	})
	if err != nil {
		return report, err
	}
	report.Valid = len(report.Errors) == 0
	s.rememberIntegrityReport(report)
	return report, nil
}
