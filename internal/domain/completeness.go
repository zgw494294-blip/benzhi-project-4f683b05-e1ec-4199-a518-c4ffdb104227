package domain

import "sort"

type CheckState string

const (
	CheckComplete   CheckState = "complete"
	CheckIncomplete CheckState = "incomplete"
	CheckBlocked    CheckState = "blocked"
)

type ReviewCheck struct {
	Code    string     `json:"code"`
	Label   string     `json:"label"`
	State   CheckState `json:"state"`
	Message string     `json:"message"`
	LotID   string     `json:"lotId,omitempty"`
}

type CompletenessReport struct {
	Ready            bool          `json:"ready"`
	Checks           []ReviewCheck `json:"checks"`
	OpenFindingCount int           `json:"openFindingCount"`
	PassedTestCount  int           `json:"passedTestCount"`
	FailedTestCount  int           `json:"failedTestCount"`
}

func (c *AccessionCase) Completeness() CompletenessReport {
	report := CompletenessReport{Ready: true, Checks: []ReviewCheck{}}
	appendCheck := func(check ReviewCheck) {
		report.Checks = append(report.Checks, check)
		if check.State != CheckComplete {
			report.Ready = false
		}
	}
	if len(c.Lots) == 0 {
		appendCheck(ReviewCheck{Code: "lots", Label: "种子批次", State: CheckIncomplete, Message: "至少登记一个批次"})
	} else {
		appendCheck(ReviewCheck{Code: "lots", Label: "种子批次", State: CheckComplete, Message: "已登记全部待检批次"})
	}
	if c.Sampling == nil {
		appendCheck(ReviewCheck{Code: "sampling", Label: "抽样方案", State: CheckIncomplete, Message: "尚未生成抽样方案"})
	} else if c.Sampling.ConfirmedAt == nil {
		appendCheck(ReviewCheck{Code: "sampling", Label: "抽样方案", State: CheckIncomplete, Message: "实际取样量尚未确认"})
	} else {
		appendCheck(ReviewCheck{Code: "sampling", Label: "抽样方案", State: CheckComplete, Message: "取样和留样边界已确认"})
	}
	for _, lot := range c.Lots {
		germination, contamination := false, false
		for _, test := range c.Tests {
			if test.LotID != lot.ID || test.IsRetest {
				continue
			}
			if test.TestType == TestGermination {
				germination = true
			}
			if test.TestType == TestContamination {
				contamination = true
			}
			if test.Result == ResultPass {
				report.PassedTestCount++
			} else {
				report.FailedTestCount++
			}
		}
		state := CheckComplete
		message := "发芽活力和污染筛查均已记录"
		if !germination || !contamination {
			state = CheckIncomplete
			message = "缺少发芽活力或污染筛查初检"
		}
		appendCheck(ReviewCheck{Code: "tests:" + lot.ID, Label: "批次 " + lot.LotCode + " 检验", State: state, Message: message, LotID: lot.ID})
	}
	for _, finding := range c.Findings {
		if finding.Status != FindingClosed {
			report.OpenFindingCount++
			appendCheck(ReviewCheck{Code: "finding:" + finding.ID, Label: "异常发现项", State: CheckBlocked, Message: finding.Category + "尚未独立复核关闭", LotID: finding.LotID})
		}
	}
	if len(c.Findings) > 0 && report.OpenFindingCount == 0 {
		appendCheck(ReviewCheck{Code: "findings", Label: "异常整改", State: CheckComplete, Message: "全部发现项均有证据、复验和独立关闭记录"})
	}
	sort.SliceStable(report.Checks, func(i, j int) bool { return report.Checks[i].Code < report.Checks[j].Code })
	return report
}

func (c *AccessionCase) LatestPassingRetest(lotID string, testType TestType) *ViabilityTest {
	var latest *ViabilityTest
	for i := range c.Tests {
		test := &c.Tests[i]
		if test.LotID != lotID || test.TestType != testType || !test.IsRetest || test.Result != ResultPass {
			continue
		}
		if latest == nil || test.PerformedAt.After(latest.PerformedAt) {
			latest = test
		}
	}
	return latest
}
