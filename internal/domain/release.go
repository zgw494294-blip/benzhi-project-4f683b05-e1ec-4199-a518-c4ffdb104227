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

// CertificateMatches reports whether two release certificates agree on every
// protected field: identifier, case binding, serial number, manifest digest,
// signed case revision, approver and issuance time.
func CertificateMatches(a, b *ReleaseCertificate) bool {
	if a == nil || b == nil {
		return false
	}
	return a.ID == b.ID && a.CaseID == b.CaseID && a.SerialNumber == b.SerialNumber && a.ManifestDigest == b.ManifestDigest && a.CaseRevision == b.CaseRevision && a.ApprovedBy == b.ApprovedBy && a.IssuedAt.Equal(b.IssuedAt)
}

// ManifestMatches reports whether two frozen manifests agree on canonical JSON,
// digest, frozen-by actor and frozen-at time.
func ManifestMatches(a, b *FrozenManifest) bool {
	if a == nil || b == nil {
		return false
	}
	return a.CanonicalJSON == b.CanonicalJSON && a.Digest == b.Digest && a.FrozenBy == b.FrozenBy && a.FrozenAt.Equal(b.FrozenAt)
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
	if c.Certificate.CaseID != c.ID {
		return false, "凭据案卷标识不一致"
	}
	if c.Certificate.ApprovedBy == "" {
		return false, "凭据缺少签发人"
	}
	return true, "凭据有效，冻结清单摘要一致"
}
