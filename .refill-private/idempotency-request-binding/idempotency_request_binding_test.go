package idempotency_request_binding_test

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestIdempotencyKeyRejectsDifferentRequests(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "idempotency.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	app := application.New(store)
	now := time.Now().UTC()

	create := func(key, code, species string) (*domain.AccessionCase, error) {
		return app.CreateCase(application.CreateCaseCommand{
			Context:       application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, IdempotencyKey: key},
			AccessionCode: code, SpeciesName: species, CollectionSite: "广西花坪",
			CollectedAt: now.Add(-2 * time.Hour), ReceivedAt: now.Add(-time.Hour), Owner: "接收员甲",
		})
	}

	first, err := create("shared-create-key", "ACC-FIRST", "银杉")
	if err != nil {
		t.Fatal(err)
	}
	accepted := make([]string, 0, 2)
	if replayed, err := create("shared-create-key", "ACC-SECOND", "珙桐"); !isConflict(err) {
		if err == nil && replayed != nil && replayed.ID == first.ID {
			accepted = append(accepted, "CreateCase返回了首个请求的缓存案卷")
		} else {
			accepted = append(accepted, "CreateCase未以CONFLICT拒绝不同请求")
		}
	}

	lotContext := application.Context{
		Actor: "接收员甲", Role: domain.RoleReceiver, ExpectedRevision: first.Revision,
		IdempotencyKey: "shared-lot-key",
	}
	withFirstLot, err := app.AddLot(first.ID, application.AddLotCommand{
		Context: lotContext, LotCode: "LOT-A", ContainerCode: "BOX-A", SeedCount: 300,
		MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	lotContext.ExpectedRevision = withFirstLot.Revision
	if replayed, err := app.AddLot(first.ID, application.AddLotCommand{
		Context: lotContext, LotCode: "LOT-B", ContainerCode: "BOX-B", SeedCount: 240,
		MoisturePercent: 7, ArrivalCondition: "完整", ReservedCount: 60,
	}); !isConflict(err) {
		if err == nil && replayed != nil && len(replayed.Lots) == 1 && replayed.Lots[0].LotCode == "LOT-A" {
			accepted = append(accepted, "AddLot把不同批次请求重放成首个批次的成功响应")
		} else {
			accepted = append(accepted, "AddLot未以CONFLICT拒绝不同请求")
		}
	}

	if len(accepted) != 0 {
		t.Fatalf("TestIdempotencyKeyRejectsDifferentRequests: 幂等键未绑定原始请求: %v", accepted)
	}
}

func isConflict(err error) bool {
	var business *domain.BusinessError
	return errors.As(err, &business) && business.Code == domain.CodeConflict
}
