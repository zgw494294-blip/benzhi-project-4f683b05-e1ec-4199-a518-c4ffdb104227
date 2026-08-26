package integrity_report_cache_stale_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func createCase(t *testing.T, store *repository.Store, id, accession, key string, now time.Time) {
	t.Helper()
	c, err := domain.NewCase(id, accession, "银杉", "花坪", now.Add(-time.Hour), now, "接收员甲", now)
	if err != nil {
		t.Fatal(err)
	}
	c.Revision = 1
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	event, err := audit.NewEvent("event-"+id, id, "接收员甲", "case.created", 1, 1, now, map[string]any{"accessionCode": accession}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(repository.Mutation{Case: c, Event: event, IdempotencyKey: key, Result: raw}); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrityReportTracksLaterCommits(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	createCase(t, store, "case-1", "ACC-1", "create-1", now)
	first, err := store.CheckIntegrity()
	if err != nil {
		t.Fatal(err)
	}
	if !first.Valid || first.CaseCount != 1 || first.AuditEventCount != 1 {
		t.Fatalf("首次完整性报告异常: %+v", first)
	}

	createCase(t, store, "case-2", "ACC-2", "create-2", now.Add(time.Minute))
	second, err := store.CheckIntegrity()
	if err != nil {
		t.Fatal(err)
	}
	if !second.Valid || second.CaseCount != 2 || second.AuditEventCount != 2 {
		t.Fatalf("后续提交未进入完整性报告: %+v", second)
	}
}
