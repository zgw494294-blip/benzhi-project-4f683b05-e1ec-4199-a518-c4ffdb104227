package case_list_nested_alias_test

import (
	"path/filepath"
	"testing"
	"time"

	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestCaseListCacheDoesNotLeakNestedMutations(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "seed-vault.db"))
	if err != nil {
		t.Fatalf("打开测试数据库: %v", err)
	}
	t.Cleanup(func() { _ = store.Close() })
	service := application.New(store)

	received := time.Now().UTC().Add(-24 * time.Hour)
	created, err := service.CreateCase(application.CreateCaseCommand{
		Context:       application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, IdempotencyKey: "private-create"},
		AccessionCode: "SV-PRIVATE-001", SpeciesName: "测试物种", CollectionSite: "测试采集地",
		CollectedAt: received.Add(-24 * time.Hour), ReceivedAt: received, Owner: "接收员甲",
	})
	if err != nil {
		t.Fatalf("创建案卷: %v", err)
	}
	_, err = service.AddLot(created.ID, application.AddLotCommand{
		Context: application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, ExpectedRevision: created.Revision, IdempotencyKey: "private-add-lot"},
		LotCode: "LOT-ORIGINAL", ContainerCode: "BOX-001", SeedCount: 200,
		MoisturePercent: 8.5, ArrivalCondition: "外观完整", ReservedCount: 80,
	})
	if err != nil {
		t.Fatalf("登记批次: %v", err)
	}

	first, err := service.ListCases(domain.StatusSampling)
	if err != nil {
		t.Fatalf("首次查询列表: %v", err)
	}
	if len(first) != 1 || len(first[0].Lots) != 1 {
		t.Fatalf("首次查询结果不完整: %#v", first)
	}
	first[0].Lots[0].LotCode = "LOT-INJECTED-BY-CALLER"

	persisted, err := store.GetCase(created.ID)
	if err != nil {
		t.Fatalf("读取持久化案卷: %v", err)
	}
	if persisted.Lots[0].LotCode != "LOT-ORIGINAL" {
		t.Fatalf("测试前提失败，持久化记录被意外修改: %q", persisted.Lots[0].LotCode)
	}

	second, err := service.ListCases(domain.StatusSampling)
	if err != nil {
		t.Fatalf("再次查询列表: %v", err)
	}
	if got := second[0].Lots[0].LotCode; got != "LOT-ORIGINAL" {
		t.Fatalf("后续列表查询泄漏了前一调用方的嵌套修改: got %q, want %q", got, "LOT-ORIGINAL")
	}
}
