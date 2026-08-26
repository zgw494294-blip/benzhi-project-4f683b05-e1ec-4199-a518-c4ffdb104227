package aggregate_cache_commit_window_test

import (
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestAggregateCacheDoesNotPublishSnapshotAcrossCommit(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "cache-window.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	c, err := domain.NewCase("case-cache-window", "ACC-CACHE-WINDOW", "银杉", "花坪", now.Add(-time.Hour), now, "接收员甲", now)
	if err != nil {
		t.Fatal(err)
	}
	c.Revision = 1
	raw, err := json.Marshal(c)
	if err != nil {
		t.Fatal(err)
	}
	event, err := audit.NewEvent("event-created", c.ID, "接收员甲", "case.created", 1, 1, now, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := store.Create(repository.Mutation{Case: c, Event: event, IdempotencyKey: "create-cache-window", Result: raw}); err != nil {
		t.Fatal(err)
	}
	cached, err := store.GetCase(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if cached.Revision != 1 {
		t.Fatalf("预热案卷缓存失败: revision=%d", cached.Revision)
	}

	mutationEntered := make(chan struct{})
	allowCommit := make(chan struct{})
	mutationDone := make(chan error, 1)
	go func() {
		_, _, mutateErr := store.MutateCase(c.ID, 1, "add-lot-cache-window", "lot.added", "接收员甲", "event-lot", now.Add(time.Minute), nil, func(current *domain.AccessionCase) error {
			if err := current.AddLot(domain.SeedLot{ID: "lot-cache-window", LotCode: "LOT-CACHE-WINDOW", ContainerCode: "BOX-CACHE-WINDOW", SeedCount: 200, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 50}); err != nil {
				return err
			}
			close(mutationEntered)
			<-allowCommit
			return nil
		})
		mutationDone <- mutateErr
	}()

	<-mutationEntered
	duringCommit, err := store.GetCase(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if duringCommit.Revision != 1 || len(duringCommit.Lots) != 0 {
		t.Fatalf("提交前的并发读取没有取得旧快照: revision=%d lots=%d", duringCommit.Revision, len(duringCommit.Lots))
	}
	close(allowCommit)
	if err := <-mutationDone; err != nil {
		t.Fatal(err)
	}

	afterCommit, err := store.GetCase(c.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterCommit.Revision != 2 || len(afterCommit.Lots) != 1 {
		t.Fatalf("提交后的查询复用了提交窗口内发布的旧快照: revision=%d lots=%d", afterCommit.Revision, len(afterCommit.Lots))
	}
}
