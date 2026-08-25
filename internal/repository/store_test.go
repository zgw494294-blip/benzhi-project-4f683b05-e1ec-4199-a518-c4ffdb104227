package repository

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
)

func TestCreateMutateAndIdempotentReplay(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	c, _ := domain.NewCase("case-1", "ACC-1", "银杉", "花坪", now.Add(-time.Hour), now, "甲", now)
	c.Revision = 1
	raw, _ := json.Marshal(c)
	event, _ := audit.NewEvent("event-1", c.ID, "甲", "case.created", 1, 1, now, map[string]any{"code": "ACC-1"}, "")
	if _, replay, err := store.Create(Mutation{Case: c, Event: event, IdempotencyKey: "key-create", Result: raw}); err != nil || replay {
		t.Fatalf("创建失败: %v", err)
	}
	result, replay, err := store.MutateCase(c.ID, 1, "key-lot", "lot.added", "甲", "event-2", now, map[string]any{"lot": "L1"}, func(c *domain.AccessionCase) error {
		return c.AddLot(domain.SeedLot{ID: "lot-1", LotCode: "L1", ContainerCode: "C1", SeedCount: 200, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 50})
	})
	if err != nil || replay {
		t.Fatalf("变更失败: %v", err)
	}
	var updated domain.AccessionCase
	if err := json.Unmarshal(result, &updated); err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || len(updated.Lots) != 1 {
		t.Fatal("变更结果不完整")
	}
	replayed, isReplay, err := store.MutateCase(c.ID, 999, "key-lot", "lot.added", "甲", "ignored", now, nil, func(*domain.AccessionCase) error { return errors.New("不应执行") })
	if err != nil || !isReplay || string(replayed) != string(result) {
		t.Fatal("幂等重放未返回稳定结果")
	}
	if _, _, err := store.MutateCase(c.ID, 1, "other", "case.changed", "甲", "e3", now, nil, func(*domain.AccessionCase) error { return nil }); !errors.Is(err, ErrRevision) {
		t.Fatal("应拒绝过期expectedRevision")
	}
	report, err := store.CheckIntegrity()
	if err != nil {
		t.Fatal(err)
	}
	if !report.Valid || report.CaseCount != 1 || report.AuditEventCount != 2 {
		t.Fatalf("完整性报告异常: %+v", report)
	}
}

func TestBusinessFailureIsIdempotentWithoutAggregateOrAuditChange(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "failure.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Now().UTC()
	c, _ := domain.NewCase("case-failure", "ACC-FAILURE", "银杉", "花坪", now.Add(-time.Hour), now, "甲", now)
	c.Revision = 1
	raw, _ := json.Marshal(c)
	event, _ := audit.NewEvent("event-failure", c.ID, "甲", "case.created", 1, 1, now, nil, "")
	if _, _, err := store.Create(Mutation{Case: c, Event: event, IdempotencyKey: "create-failure", Result: raw}); err != nil {
		t.Fatal(err)
	}
	invalidLots := []domain.SeedLot{
		{ID: "l1", LotCode: "SAME", ContainerCode: "C1", SeedCount: 200, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 50},
		{ID: "l2", LotCode: "same", ContainerCode: "C2", SeedCount: 200, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 50},
	}
	_, replay, firstErr := store.MutateCase(c.ID, 1, "batch-failure", "lots.batch_added", "甲", "event-unused", now, nil, func(current *domain.AccessionCase) error { return current.AddLots(invalidLots) })
	if firstErr == nil || replay {
		t.Fatal("首次无效批量请求应返回业务错误")
	}
	called := false
	_, replay, secondErr := store.MutateCase(c.ID, 999, "batch-failure", "lots.batch_added", "甲", "event-unused-2", now, nil, func(*domain.AccessionCase) error { called = true; return nil })
	if secondErr == nil || !replay || called || secondErr.Error() != firstErr.Error() {
		t.Fatal("无效结果未被稳定幂等重放")
	}
	stored, err := store.GetCase(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	events, err := store.Timeline(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Revision != 1 || len(stored.Lots) != 0 || len(events) != 1 {
		t.Fatal("失败批量请求改变了聚合、版本或审计时间线")
	}
}
