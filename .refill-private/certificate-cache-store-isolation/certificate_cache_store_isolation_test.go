package certificate_cache_store_isolation_test

import (
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestCertificateCacheIsScopedToStoreLifecycle(t *testing.T) {
	first, err := repository.Open(filepath.Join(t.TempDir(), "first.db"))
	if err != nil {
		t.Fatal(err)
	}
	firstCertificate := &domain.ReleaseCertificate{
		ID:             "certificate-first",
		CaseID:         "case-first",
		ManifestDigest: "digest-first",
		CaseRevision:   9,
		ApprovedBy:     "审核员甲",
		IssuedAt:       time.Unix(1_700_000_000, 0).UTC(),
	}
	firstManifest := &domain.FrozenManifest{
		CanonicalJSON: `{"case":"first"}`,
		Digest:        firstCertificate.ManifestDigest,
		FrozenBy:      firstCertificate.ApprovedBy,
		FrozenAt:      firstCertificate.IssuedAt,
	}
	if err := first.Update(func(tx *bolt.Tx) error {
		_, err := repository.AllocateCertificate(tx, firstCertificate, firstManifest)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if firstCertificate.SerialNumber != 1 {
		t.Fatalf("首个数据库的凭据序号=%d，期望1", firstCertificate.SerialNumber)
	}
	if _, err := first.CertificateBySerial(1); err != nil {
		t.Fatal(err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}

	second, err := repository.Open(filepath.Join(t.TempDir(), "second.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	secondCertificate := &domain.ReleaseCertificate{
		ID:             "certificate-second",
		CaseID:         "case-second",
		ManifestDigest: "digest-second",
		CaseRevision:   4,
		ApprovedBy:     "审核员乙",
		IssuedAt:       time.Unix(1_800_000_000, 0).UTC(),
	}
	secondManifest := &domain.FrozenManifest{
		CanonicalJSON: `{"case":"second"}`,
		Digest:        secondCertificate.ManifestDigest,
		FrozenBy:      secondCertificate.ApprovedBy,
		FrozenAt:      secondCertificate.IssuedAt,
	}
	if err := second.Update(func(tx *bolt.Tx) error {
		_, err := repository.AllocateCertificate(tx, secondCertificate, secondManifest)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	got, err := second.CertificateBySerial(1)
	if err != nil {
		t.Fatal(err)
	}
	if got.CaseID != secondCertificate.CaseID || got.ManifestDigest != secondCertificate.ManifestDigest {
		t.Fatalf("TestCertificateCacheIsScopedToStoreLifecycle: 第二个数据库返回了旧凭据 caseID=%q digest=%q", got.CaseID, got.ManifestDigest)
	}
}
