package detailactionscachealias_test

import (
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestCaseDetailActionsAreIsolatedAcrossQueries(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "detail-actions.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })

	service := application.New(store)
	now := time.Date(2026, time.August, 25, 9, 0, 0, 0, time.UTC)
	created, err := service.CreateCase(application.CreateCaseCommand{
		Context: application.Context{
			Actor:          "接收员甲",
			Role:           domain.RoleReceiver,
			IdempotencyKey: "create-detail-actions-alias",
		},
		AccessionCode:  "ALIAS-2026-001",
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
	wantFirstAction := first.AllowedActions[domain.RoleReceiver][0]
	first.AllowedActions[domain.RoleReceiver][0] = "调用方注入的伪造动作"
	delete(first.AllowedActions, domain.RoleReviewer)

	second, err := service.GetCase(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	receiverActions := second.AllowedActions[domain.RoleReceiver]
	_, hasReviewer := second.AllowedActions[domain.RoleReviewer]
	if len(receiverActions) == 0 || receiverActions[0] != wantFirstAction || !hasReviewer {
		t.Fatalf("详情动作被先前查询结果污染: receiver=%v reviewerPresent=%v", receiverActions, hasReviewer)
	}
}
