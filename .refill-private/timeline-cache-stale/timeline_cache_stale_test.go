package timeline_cache_stale_test

import (
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestDetailTimelineCacheTracksCommittedRevision(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "timeline-cache.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store)
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	created, err := service.CreateCase(application.CreateCaseCommand{
		Context: application.Context{
			Actor:          "接收员甲",
			Role:           domain.RoleReceiver,
			IdempotencyKey: "cache-create",
		},
		AccessionCode:  "CACHE-2026-001",
		SpeciesName:    "银杉",
		CollectionSite: "广西花坪保护区",
		CollectedAt:    now.Add(-48 * time.Hour),
		ReceivedAt:     now.Add(-24 * time.Hour),
		Owner:          "接收员甲",
	})
	if err != nil {
		t.Fatal(err)
	}
	first, err := service.GetCase(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Timeline) != 1 || first.Timeline[0].CaseRevision != created.Revision {
		t.Fatalf("初次详情时间线与案卷版本不一致: revision=%d events=%d", created.Revision, len(first.Timeline))
	}

	updated, err := service.AddLot(created.ID, application.AddLotCommand{
		Context: application.Context{
			Actor:            "接收员甲",
			Role:             domain.RoleReceiver,
			ExpectedRevision: created.Revision,
			IdempotencyKey:   "cache-add-lot",
		},
		LotCode:          "LOT-CACHE-1",
		ContainerCode:    "BOX-CACHE-1",
		SeedCount:        200,
		MoisturePercent:  7.5,
		ArrivalCondition: "完整、干燥",
		ReservedCount:    50,
	})
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.GetCase(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Timeline) != 2 || second.Timeline[len(second.Timeline)-1].CaseRevision != updated.Revision {
		t.Fatalf("提交后详情仍返回旧时间线: caseRevision=%d timelineEvents=%d", second.Case.Revision, len(second.Timeline))
	}
}
