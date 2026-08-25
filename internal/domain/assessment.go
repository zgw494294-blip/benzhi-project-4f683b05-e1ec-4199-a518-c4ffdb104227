package domain

import "fmt"

type TestAssessment struct {
	TestID            string     `json:"testId"`
	LotID             string     `json:"lotId"`
	TestType          TestType   `json:"testType"`
	TestedCount       int        `json:"testedCount"`
	PassedCount       int        `json:"passedCount"`
	ContaminatedCount int        `json:"contaminatedCount"`
	PassRate          float64    `json:"passRate"`
	ContaminationRate float64    `json:"contaminationRate"`
	Threshold         float64    `json:"threshold"`
	Comparator        string     `json:"comparator"`
	Result            TestResult `json:"result"`
	Conclusion        string     `json:"conclusion"`
}

func AssessTest(test ViabilityTest) (TestAssessment, error) {
	if test.TestedCount < 1 {
		return TestAssessment{}, Invalid("检验%s的受检总数无效", test.ID)
	}
	if test.PassedCount < 0 || test.ContaminatedCount < 0 || test.PassedCount > test.TestedCount || test.ContaminatedCount > test.TestedCount {
		return TestAssessment{}, Invalid("检验%s的计数关系无效", test.ID)
	}
	passRate := float64(test.PassedCount) / float64(test.TestedCount) * 100
	contaminationRate := float64(test.ContaminatedCount) / float64(test.TestedCount) * 100
	assessment := TestAssessment{
		TestID:            test.ID,
		LotID:             test.LotID,
		TestType:          test.TestType,
		TestedCount:       test.TestedCount,
		PassedCount:       test.PassedCount,
		ContaminatedCount: test.ContaminatedCount,
		PassRate:          passRate,
		ContaminationRate: contaminationRate,
		Threshold:         test.Threshold,
	}
	switch test.TestType {
	case TestGermination:
		assessment.Comparator = ">="
		if passRate >= test.Threshold {
			assessment.Result = ResultPass
			assessment.Conclusion = fmt.Sprintf("发芽率%.2f%%达到下限%.2f%%", passRate, test.Threshold)
		} else {
			assessment.Result = ResultFail
			assessment.Conclusion = fmt.Sprintf("发芽率%.2f%%低于下限%.2f%%", passRate, test.Threshold)
		}
	case TestContamination:
		assessment.Comparator = "<="
		if contaminationRate <= test.Threshold {
			assessment.Result = ResultPass
			assessment.Conclusion = fmt.Sprintf("污染率%.2f%%未超过上限%.2f%%", contaminationRate, test.Threshold)
		} else {
			assessment.Result = ResultFail
			assessment.Conclusion = fmt.Sprintf("污染率%.2f%%超过上限%.2f%%", contaminationRate, test.Threshold)
		}
	default:
		return TestAssessment{}, Invalid("检验%s的类型无效", test.ID)
	}
	return assessment, nil
}

func (c *AccessionCase) Assessments() ([]TestAssessment, error) {
	result := make([]TestAssessment, 0, len(c.Tests))
	for _, test := range c.Tests {
		assessment, err := AssessTest(test)
		if err != nil {
			return nil, err
		}
		result = append(result, assessment)
	}
	return result, nil
}
