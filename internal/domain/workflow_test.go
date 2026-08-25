package domain

import (
	"testing"
	"time"
)

func TestCompleteWorkflowWithFinding(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	c, err := NewCase("case-1", "ACC-1", "银杉", "广西花坪", now.Add(-48*time.Hour), now.Add(-24*time.Hour), "接收员甲", now)
	if err != nil {
		t.Fatal(err)
	}
	lot := SeedLot{ID: "lot-1", LotCode: "LOT-1", ContainerCode: "C-1", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80}
	if err := c.AddLot(lot); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusSampling {
		t.Fatalf("添加批次后状态=%s", c.Status)
	}
	if err := c.GenerateSamplingPlan("sample-1"); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfirmSampling(map[string]int{"lot-1": 50}, "接收员甲", now); err != nil {
		t.Fatal(err)
	}
	_, finding, err := c.RecordTest(TestInput{ID: "test-1", LotID: "lot-1", Type: TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 35, Threshold: 80, Actor: "检验员乙", At: now})
	if err != nil {
		t.Fatal(err)
	}
	if finding == nil || c.Status != StatusRemediation {
		t.Fatal("异常检验未创建发现项")
	}
	_, _, err = c.RecordTest(TestInput{ID: "test-2", LotID: "lot-1", Type: TestContamination, Replicates: []int{25, 25}, Tested: 50, Passed: 49, Contaminated: 1, Threshold: 5, Actor: "检验员乙", At: now})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = c.RecordTest(TestInput{ID: "test-3", LotID: "lot-1", Type: TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 46, Threshold: 80, Actor: "检验员乙", At: now, IsRetest: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.RemediateFinding(finding.ID, "影像证据EV-1", "检验员乙", "test-3"); err != nil {
		t.Fatal(err)
	}
	if err := c.CloseFinding(finding.ID, "检验员乙", RoleTester, now); err == nil {
		t.Fatal("同一整改人不应能关闭发现项")
	}
	if err := c.CloseFinding(finding.ID, "检验员丙", RoleTester, now); err != nil {
		t.Fatal(err)
	}
	if c.Status != StatusReview || !c.Completeness().Ready {
		t.Fatalf("案卷未进入待审核: %s", c.Status)
	}
	if err := c.Freeze("审核员丁", RoleReviewer, now); err != nil {
		t.Fatal(err)
	}
	cert, err := c.IssueCertificate("cert-1", 1, "审核员丁", now)
	if err != nil {
		t.Fatal(err)
	}
	if cert.ManifestDigest == "" || c.Status != StatusReleased {
		t.Fatal("凭据未正确签发")
	}
	if valid, message := VerifyCertificate(c); !valid {
		t.Fatalf("凭据应有效: %s", message)
	}
}

func TestSamplingAndTestValidation(t *testing.T) {
	now := time.Now().UTC()
	c, _ := NewCase("c", "A", "物种", "地点", now.Add(-time.Hour), now, "甲", now)
	if err := c.AddLot(SeedLot{ID: "l", LotCode: "L", ContainerCode: "C", SeedCount: 100, MoisturePercent: 5, ArrivalCondition: "好", ReservedCount: 90}); err != nil {
		t.Fatal(err)
	}
	if err := c.GenerateSamplingPlan("s"); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfirmSampling(map[string]int{"l": 20}, "甲", now); err == nil {
		t.Fatal("超出留样边界的取样应失败")
	}
	c.Sampling.Items[0].PlannedCount = 10
	if err := c.ConfirmSampling(map[string]int{"l": 10}, "甲", now); err != nil {
		t.Fatal(err)
	}
	_, _, err := c.RecordTest(TestInput{ID: "t", LotID: "l", Type: TestGermination, Replicates: []int{10, 10}, Tested: 19, Passed: 18, Threshold: 80, Actor: "乙", At: now})
	if err == nil {
		t.Fatal("重复计数与受检总数不一致时应失败")
	}
}

func TestExtendedAtomicWorkflowBoundaries(t *testing.T) {
	now := time.Date(2026, 8, 25, 9, 0, 0, 0, time.UTC)
	c, err := NewCase("case-extended", "ACC-EXT", "银杉", "广西花坪", now.Add(-48*time.Hour), now.Add(-24*time.Hour), "接收员甲", now)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.ReviseBaseData("银杉", "广西花坪核心区", now.Add(-48*time.Hour), now.Add(-24*time.Hour), "接收员甲"); err != nil {
		t.Fatal(err)
	}
	invalid := []SeedLot{
		{ID: "lot-x", LotCode: "LOT-X", ContainerCode: "CX", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80},
		{ID: "lot-y", LotCode: "lot-x", ContainerCode: "CY", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80},
	}
	if err := c.AddLots(invalid); err == nil || len(c.Lots) != 0 {
		t.Fatal("批量重复编号应整批拒绝")
	}
	lots := []SeedLot{
		{ID: "lot-a", LotCode: "LOT-A", ContainerCode: "CA", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80},
		{ID: "lot-b", LotCode: "LOT-B", ContainerCode: "CB", SeedCount: 300, MoisturePercent: 8, ArrivalCondition: "完整", ReservedCount: 80},
	}
	if err := c.AddLots(lots); err != nil {
		t.Fatal(err)
	}
	if err := c.GenerateSamplingPlan("sampling-extended"); err != nil {
		t.Fatal(err)
	}
	if err := c.ConfirmSamplingWithDeviations(map[string]int{"lot-a": 50, "lot-b": 50, "unknown": 50}, nil, "接收员甲", now); err == nil || c.Sampling.ConfirmedAt != nil {
		t.Fatal("未知批次应拒绝整次抽样确认")
	}
	if err := c.ConfirmSamplingWithDeviations(map[string]int{"lot-a": 55, "lot-b": 50}, nil, "接收员甲", now); err == nil {
		t.Fatal("偏离计划时必须填写原因")
	}
	if err := c.ConfirmSamplingWithDeviations(map[string]int{"lot-a": 55, "lot-b": 50}, map[string]string{"lot-a": "器皿损耗补量"}, "接收员甲", now); err != nil {
		t.Fatal(err)
	}
	if c.Sampling.Items[0].AvailableCount != 165 || c.Sampling.Items[0].DeviationReason == "" {
		t.Fatal("抽样平衡或偏差说明未冻结")
	}
	if _, err := c.ReviseBaseData("银杉", "其他地点", c.CollectedAt, c.ReceivedAt, c.Owner); err == nil {
		t.Fatal("抽样确认后不应允许修订基础资料")
	}
	inputs := []TestInput{
		{ID: "ta-g", LotID: "lot-a", Type: TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 45, Threshold: 80, Actor: "检验员乙", At: now},
		{ID: "ta-c", LotID: "lot-a", Type: TestContamination, Replicates: []int{25, 25}, Tested: 50, Passed: 49, Contaminated: 1, Threshold: 5, Actor: "检验员乙", At: now},
		{ID: "tb-g", LotID: "lot-b", Type: TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 45, Threshold: 80, Actor: "检验员乙", At: now},
		{ID: "tb-c", LotID: "lot-b", Type: TestContamination, Replicates: []int{25, 25}, Tested: 50, Passed: 45, Contaminated: 5, Threshold: 5, Actor: "检验员乙", At: now},
	}
	if _, _, err := c.RecordTests(inputs); err != nil {
		t.Fatal(err)
	}
	matrix, err := c.TestMatrix()
	if err != nil || len(matrix) != 4 {
		t.Fatalf("检验矩阵不完整: %v", err)
	}
	if len(c.Findings) != 1 || c.Status != StatusRemediation {
		t.Fatal("异常单元应单独生成发现项")
	}
	finding := &c.Findings[0]
	if err := c.AssignFinding(finding.ID, "检验员乙", now.Add(7*24*time.Hour)); err != nil {
		t.Fatal(err)
	}
	for i, description := range []string{"完成消毒", "复核培养基"} {
		evidence := FindingEvidence{ID: string(rune('a' + i)), SubmittedBy: "检验员乙", Description: description, ContentDigest: DigestText(description), SubmittedAt: now.Add(time.Duration(i) * time.Hour)}
		if err := c.AddFindingEvidence(finding.ID, evidence); err != nil {
			t.Fatal(err)
		}
	}
	if len(finding.EvidenceLedger) != 2 {
		t.Fatal("证据台账必须保持追加顺序")
	}
	if _, _, err := c.RecordTest(TestInput{ID: "tb-retest-wrong", LotID: "lot-b", Type: TestGermination, Replicates: []int{25, 25}, Tested: 50, Passed: 45, Threshold: 80, Actor: "检验员乙", At: now, IsRetest: true}); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitFindingRemediation(finding.ID, "检验员乙", "tb-retest-wrong"); err == nil {
		t.Fatal("不同类型复验不应匹配发现项")
	}
	if _, _, err := c.RecordTest(TestInput{ID: "tb-retest", LotID: "lot-b", Type: TestContamination, Replicates: []int{25, 25}, Tested: 50, Passed: 49, Contaminated: 1, Threshold: 5, Actor: "检验员乙", At: now, IsRetest: true}); err != nil {
		t.Fatal(err)
	}
	if err := c.SubmitFindingRemediation(finding.ID, "检验员乙", "tb-retest"); err != nil {
		t.Fatal(err)
	}
	if err := c.CloseFinding(finding.ID, "检验员乙", RoleTester, now); err == nil {
		t.Fatal("整改负责人不应关闭自己的发现项")
	}
	if err := c.CloseFinding(finding.ID, "检验员丙", RoleTester, now); err != nil {
		t.Fatal(err)
	}
}
