package orphan_certificate_on_conflict_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/httpui"
	"seed-vault-release/internal/repository"
)

func readyCase(t *testing.T, now time.Time) *domain.AccessionCase {
	t.Helper()
	c, err := domain.NewCase("case-conflict", "ACC-CONFLICT", "银杉", "广西花坪", now.Add(-48*time.Hour), now.Add(-24*time.Hour), "接收员甲", now)
	if err != nil {
		t.Fatal(err)
	}
	if err := c.AddLot(domain.SeedLot{ID: "lot-1", LotCode: "LOT-1", ContainerCode: "C-1", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80}); err != nil {
		t.Fatal(err)
	}
	if err := c.GenerateSamplingPlan("sampling-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfirmSampling(map[string]int{"lot-1": 50}, "接收员甲", now); err != nil {
		t.Fatal(err)
	}
	inputs := []domain.TestInput{
		{ID: "test-g", LotID: "lot-1", Type: domain.TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 45, Threshold: 80, Actor: "检验员乙", At: now},
		{ID: "test-c", LotID: "lot-1", Type: domain.TestContamination, Replicates: []int{25, 25}, Tested: 50, Passed: 49, Contaminated: 1, Threshold: 5, Actor: "检验员乙", At: now},
	}
	if _, _, err := c.RecordTests(inputs); err != nil {
		t.Fatal(err)
	}
	if c.Status != domain.StatusReview || !c.Completeness().Ready {
		t.Fatalf("测试案卷未进入待审核状态: %s", c.Status)
	}
	return c
}

func TestRevisionConflictDoesNotLeaveOrphanCertificate(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	store, err := repository.Open(filepath.Join(t.TempDir(), "orphan.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	c := readyCase(t, now)
	c.Revision = 1
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	event, err := audit.NewEvent("event-created", c.ID, "接收员甲", "case.created", 1, 1, now, map[string]any{"accessionCode": c.AccessionCode}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(repository.Mutation{Case: c, Event: event, IdempotencyKey: "create-conflict", Result: raw}); err != nil {
		t.Fatal(err)
	}

	server := httptest.NewServer(httpui.New(application.New(store)).Handler())
	defer server.Close()
	body := bytes.NewBufferString(`{"actor":"审核员丁","role":"reviewer","expectedRevision":0,"idempotencyKey":"approve-stale"}`)
	response, err := http.Post(server.URL+"/api/v1/cases/"+c.ID+"/approve", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("过期审核应返回409，实际%d", response.StatusCode)
	}

	stored, err := store.GetCase(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision != 1 || stored.Status != domain.StatusReview || stored.Certificate != nil {
		t.Fatalf("冲突审核不应改变案卷: revision=%d status=%s certificate=%v", stored.Revision, stored.Status, stored.Certificate)
	}
	if _, err := store.CertificateBySerial(1); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("冲突审核遗留了孤儿凭据 #1: %v", err)
	}
	if _, err := store.ManifestBySerial(1); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("冲突审核遗留了孤儿冻结清单 #1: %v", err)
	}
}
