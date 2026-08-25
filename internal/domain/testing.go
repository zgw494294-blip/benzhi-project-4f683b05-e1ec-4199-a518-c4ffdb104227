package domain

import "time"

type TestInput struct {
	ID           string
	LotID        string
	Type         TestType
	Replicates   []int
	Tested       int
	Passed       int
	Contaminated int
	Threshold    float64
	Actor        string
	At           time.Time
	IsRetest     bool
}

func (c *AccessionCase) RecordTest(in TestInput) (*ViabilityTest, *Finding, error) {
	tests, findings, err := c.RecordTests([]TestInput{in})
	if err != nil {
		return nil, nil, err
	}
	var finding *Finding
	if len(findings) > 0 {
		finding = findings[0]
	}
	return tests[0], finding, nil
}

func (c *AccessionCase) RecordTests(inputs []TestInput) ([]*ViabilityTest, []*Finding, error) {
	if err := c.EnsureMutable(); err != nil {
		return nil, nil, err
	}
	if c.Status != StatusTesting && c.Status != StatusRemediation {
		return nil, nil, Conflict("当前状态不能记录检验")
	}
	if len(inputs) == 0 {
		return nil, nil, Invalid("批量检验至少包含一项结果")
	}
	existingIDs := make(map[string]struct{}, len(c.Tests))
	existingInitial := make(map[string]struct{}, len(c.Tests))
	for _, test := range c.Tests {
		existingIDs[test.ID] = struct{}{}
		if !test.IsRetest {
			existingInitial[test.LotID+"\x00"+string(test.TestType)] = struct{}{}
		}
	}
	requestInitial := map[string]struct{}{}
	prepared := make([]ViabilityTest, len(inputs))
	for i, in := range inputs {
		lot, err := c.Lot(in.LotID)
		if err != nil {
			return nil, nil, Invalid("第%d项: 批次不存在", i+1)
		}
		if err := ValidateTestInput(in); err != nil {
			return nil, nil, Invalid("第%d项（批次%s/%s）: %v", i+1, lot.LotCode, in.Type, err)
		}
		if _, exists := existingIDs[in.ID]; exists {
			return nil, nil, Conflict("第%d项: 检验标识重复", i+1)
		}
		existingIDs[in.ID] = struct{}{}
		pair := in.LotID + "\x00" + string(in.Type)
		if !in.IsRetest {
			if _, exists := existingInitial[pair]; exists {
				return nil, nil, Conflict("第%d项（批次%s/%s）: 已有同类型初检", i+1, lot.LotCode, in.Type)
			}
			if _, exists := requestInitial[pair]; exists {
				return nil, nil, Conflict("第%d项（批次%s/%s）: 请求内同类型初检重复", i+1, lot.LotCode, in.Type)
			}
			requestInitial[pair] = struct{}{}
		}
		test := ViabilityTest{ID: in.ID, CaseID: c.ID, LotID: in.LotID, TestType: in.Type, ReplicateCounts: append([]int(nil), in.Replicates...), TestedCount: in.Tested, PassedCount: in.Passed, ContaminatedCount: in.Contaminated, Threshold: in.Threshold, Result: ResultPass, PerformedBy: in.Actor, PerformedAt: in.At.UTC(), IsRetest: in.IsRetest}
		assessment, err := AssessTest(test)
		if err != nil {
			return nil, nil, Invalid("第%d项（批次%s/%s）: %v", i+1, lot.LotCode, in.Type, err)
		}
		test.Result = assessment.Result
		prepared[i] = test
	}
	testStart, findingStart := len(c.Tests), len(c.Findings)
	for _, test := range prepared {
		c.Tests = append(c.Tests, test)
		if test.Result == ResultFail && !test.IsRetest {
			category := "污染率超出阈值"
			if test.TestType == TestGermination {
				category = "发芽率未达阈值"
			}
			c.Findings = append(c.Findings, Finding{ID: "finding-" + test.ID, CaseID: c.ID, LotID: test.LotID, SourceTestID: test.ID, Category: category, Severity: "major", Status: FindingOpen, CreatedAt: test.PerformedAt, EvidenceLedger: []FindingEvidence{}})
		}
	}
	c.RefreshStatus()
	tests := make([]*ViabilityTest, 0, len(prepared))
	for i := testStart; i < len(c.Tests); i++ {
		tests = append(tests, &c.Tests[i])
	}
	findings := make([]*Finding, 0, len(c.Findings)-findingStart)
	for i := findingStart; i < len(c.Findings); i++ {
		findings = append(findings, &c.Findings[i])
	}
	return tests, findings, nil
}

type TestMatrixCell struct {
	LotID      string          `json:"lotId"`
	LotCode    string          `json:"lotCode"`
	TestType   TestType        `json:"testType"`
	State      string          `json:"state"`
	Test       *ViabilityTest  `json:"test,omitempty"`
	Assessment *TestAssessment `json:"assessment,omitempty"`
}

func (c *AccessionCase) TestMatrix() ([]TestMatrixCell, error) {
	cells := make([]TestMatrixCell, 0, len(c.Lots)*2)
	for _, lot := range c.Lots {
		for _, testType := range []TestType{TestGermination, TestContamination} {
			cell := TestMatrixCell{LotID: lot.ID, LotCode: lot.LotCode, TestType: testType, State: "missing"}
			for i := range c.Tests {
				test := &c.Tests[i]
				if test.LotID != lot.ID || test.TestType != testType || test.IsRetest {
					continue
				}
				assessment, err := AssessTest(*test)
				if err != nil {
					return nil, err
				}
				cell.Test, cell.Assessment = test, &assessment
				if test.Result == ResultPass {
					cell.State = "pass"
				} else {
					cell.State = "abnormal"
				}
				break
			}
			cells = append(cells, cell)
		}
	}
	return cells, nil
}

func ValidateTestInput(in TestInput) error {
	if in.Type != TestGermination && in.Type != TestContamination {
		return Invalid("不支持的检验类型")
	}
	if err := RequireText("检验员", in.Actor); err != nil {
		return err
	}
	if len(in.Replicates) < 2 || len(in.Replicates) > 8 {
		return Invalid("重复次数必须在2到8之间")
	}
	sum := 0
	for _, count := range in.Replicates {
		if count < 1 {
			return Invalid("每组重复计数必须大于0")
		}
		sum += count
	}
	if in.Tested != sum {
		return Invalid("受检总数必须等于各重复计数之和")
	}
	if in.Passed < 0 || in.Contaminated < 0 || in.Passed > in.Tested || in.Contaminated > in.Tested {
		return Invalid("通过、污染计数不得超出受检总数")
	}
	if in.Passed+in.Contaminated > in.Tested {
		return Invalid("通过与污染计数关系不合法")
	}
	if in.Threshold < 0 || in.Threshold > 100 {
		return Invalid("阈值必须在0到100之间")
	}
	return nil
}
