package repository

import (
	"encoding/json"
	"fmt"
	"strconv"

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

func (s *Store) CheckIntegrity() (IntegrityReport, error) {
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
				storedCertBytes := tx.Bucket(bucketCertificates).Get(certificateKey(c.Certificate.SerialNumber))
				if storedCertBytes == nil {
					report.Errors = append(report.Errors, "已放行案卷"+c.ID+"缺少只追加凭据记录")
				} else {
					var storedCert domain.ReleaseCertificate
					if err := json.Unmarshal(storedCertBytes, &storedCert); err != nil {
						report.Errors = append(report.Errors, "已放行案卷"+c.ID+"只追加凭据记录无法解码")
					} else if !domain.CertificateMatches(&storedCert, c.Certificate) {
						report.Errors = append(report.Errors, "已放行案卷"+c.ID+"只追加凭据记录与案卷内凭据不一致")
					}
				}
				manifestKey := []byte(strconv.FormatUint(c.Certificate.SerialNumber, 10))
				storedManifestBytes := tx.Bucket(bucketManifests).Get(manifestKey)
				if storedManifestBytes == nil {
					report.Errors = append(report.Errors, "已放行案卷"+c.ID+"缺少只追加冻结清单记录")
				} else {
					var storedManifest domain.FrozenManifest
					if err := json.Unmarshal(storedManifestBytes, &storedManifest); err != nil {
						report.Errors = append(report.Errors, "已放行案卷"+c.ID+"只追加冻结清单记录无法解码")
					} else if !domain.ManifestMatches(&storedManifest, c.FrozenManifest) {
						report.Errors = append(report.Errors, "已放行案卷"+c.ID+"只追加冻结清单记录与案卷内清单不一致")
					}
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
	return report, nil
}
