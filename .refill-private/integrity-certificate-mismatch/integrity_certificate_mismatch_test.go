package integrity_certificate_mismatch_test

import (
	"encoding/binary"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/application"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func TestIntegrityRejectsMismatchedAppendOnlyCertificate(t *testing.T) {
	store, err := repository.Open(filepath.Join(t.TempDir(), "integrity.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	app := application.New(store)
	now := time.Now().UTC()

	c, err := app.CreateCase(application.CreateCaseCommand{
		Context:       application.Context{Actor: "接收员甲", Role: domain.RoleReceiver, IdempotencyKey: "create"},
		AccessionCode: "ACC-INTEGRITY", SpeciesName: "银杉", CollectionSite: "广西花坪",
		CollectedAt: now.Add(-2 * time.Hour), ReceivedAt: now.Add(-time.Hour), Owner: "接收员甲",
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = app.AddLot(c.ID, application.AddLotCommand{
		Context: mutationContext("接收员甲", domain.RoleReceiver, c.Revision, "lot"),
		LotCode: "LOT-A", ContainerCode: "BOX-A", SeedCount: 300, MoisturePercent: 8,
		ArrivalCondition: "完整", ReservedCount: 80,
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = app.GenerateSampling(c.ID, mutationContext("接收员甲", domain.RoleReceiver, c.Revision, "sampling"))
	if err != nil {
		t.Fatal(err)
	}
	c, err = app.ConfirmSampling(c.ID, application.ConfirmSamplingCommand{
		Context:      mutationContext("接收员甲", domain.RoleReceiver, c.Revision, "confirm"),
		ActualCounts: map[string]int{c.Lots[0].ID: c.Sampling.Items[0].PlannedCount},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = app.RecordTests(c.ID, application.RecordTestsCommand{
		Context: mutationContext("检验员乙", domain.RoleTester, c.Revision, "tests"),
		Items: []application.TestResultInput{
			{LotID: c.Lots[0].ID, TestType: domain.TestGermination, ReplicateCounts: []int{25, 25}, TestedCount: 50, PassedCount: 45, Threshold: 80},
			{LotID: c.Lots[0].ID, TestType: domain.TestContamination, ReplicateCounts: []int{25, 25}, TestedCount: 50, PassedCount: 49, ContaminatedCount: 1, Threshold: 5},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	c, err = app.Approve(c.ID, application.ApproveCommand{Context: mutationContext("审核员丁", domain.RoleReviewer, c.Revision, "approve")})
	if err != nil {
		t.Fatal(err)
	}
	if c.Certificate == nil {
		t.Fatal("测试准备失败：未签发凭据")
	}
	baseline, err := store.CheckIntegrity()
	if err != nil || !baseline.Valid {
		t.Fatalf("测试准备失败：基线完整性异常: %+v, %v", baseline, err)
	}

	tampered := *c.Certificate
	tampered.ApprovedBy = "伪造审核员"
	encoded, err := json.Marshal(&tampered)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Update(func(tx *bolt.Tx) error {
		key := make([]byte, 8)
		binary.BigEndian.PutUint64(key, tampered.SerialNumber)
		return tx.Bucket([]byte("certificates")).Put(key, encoded)
	}); err != nil {
		t.Fatal(err)
	}

	report, err := store.CheckIntegrity()
	if err != nil {
		t.Fatal(err)
	}
	if report.Valid {
		t.Fatalf("TestIntegrityRejectsMismatchedAppendOnlyCertificate: 只追加凭据与案卷内凭据不一致却仍报告Valid=true")
	}
}

func mutationContext(actor string, role domain.Role, revision uint64, key string) application.Context {
	return application.Context{Actor: actor, Role: role, ExpectedRevision: revision, IdempotencyKey: key}
}
