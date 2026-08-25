package domain

import "time"

func (c *AccessionCase) IssueCertificate(id string, serial uint64, actor string, now time.Time) (*ReleaseCertificate, error) {
	if c.FrozenManifest == nil {
		return nil, Conflict("案卷尚未冻结")
	}
	if c.Certificate != nil {
		return nil, Conflict("案卷已经签发凭据")
	}
	if serial == 0 {
		return nil, Invalid("凭据序号必须大于0")
	}
	cert := &ReleaseCertificate{ID: id, CaseID: c.ID, SerialNumber: serial, ManifestDigest: c.FrozenManifest.Digest, CaseRevision: c.Revision + 1, ApprovedBy: actor, IssuedAt: now.UTC()}
	c.Certificate = cert
	c.Status = StatusReleased
	return cert, nil
}

func VerifyCertificate(c *AccessionCase) (bool, string) {
	if c.Certificate == nil || c.FrozenManifest == nil {
		return false, "凭据或冻结清单不存在"
	}
	_, digest, err := BuildManifest(c)
	if err != nil {
		return false, "无法重建清单摘要"
	}
	if digest != c.FrozenManifest.Digest || digest != c.Certificate.ManifestDigest {
		return false, "清单摘要不一致"
	}
	if c.Status != StatusReleased {
		return false, "案卷并非已放行状态"
	}
	return true, "凭据有效，冻结清单摘要一致"
}
