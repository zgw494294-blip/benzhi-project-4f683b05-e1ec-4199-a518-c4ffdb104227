package application

import (
	"seed-vault-release/internal/domain"
	"time"
)

// 以下 requestPayload 构建器为幂等请求指纹提供稳定输入：仅保留目标、调用身份与业务载荷，
// 排除 idempotencyKey 与服务端生成的标识/时间戳，确保同一请求重试指纹稳定一致，
// 而目标、业务载荷或调用身份任一不同都会得到不同指纹并触发 CONFLICT。

type identityPayload struct {
	Actor            string      `json:"actor"`
	Role             domain.Role `json:"role"`
	ExpectedRevision uint64      `json:"expectedRevision"`
}

func createCasePayload(ctx Context, cmd CreateCaseCommand) any {
	return struct {
		identityPayload
		AccessionCode  string    `json:"accessionCode"`
		SpeciesName    string    `json:"speciesName"`
		CollectionSite string    `json:"collectionSite"`
		CollectedAt    time.Time `json:"collectedAt"`
		ReceivedAt     time.Time `json:"receivedAt"`
		Owner          string    `json:"owner"`
	}{identityPayload{ctx.Actor, ctx.Role, ctx.ExpectedRevision}, cmd.AccessionCode, cmd.SpeciesName, cmd.CollectionSite, cmd.CollectedAt, cmd.ReceivedAt, cmd.Owner}
}

func addLotPayload(cmd AddLotCommand) any {
	return struct {
		identityPayload
		LotCode          string  `json:"lotCode"`
		ContainerCode    string  `json:"containerCode"`
		SeedCount        int     `json:"seedCount"`
		MoisturePercent  float64 `json:"moisturePercent"`
		ArrivalCondition string  `json:"arrivalCondition"`
		ReservedCount    int     `json:"reservedCount"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.LotCode, cmd.ContainerCode, cmd.SeedCount, cmd.MoisturePercent, cmd.ArrivalCondition, cmd.ReservedCount}
}

func reviseCasePayload(cmd ReviseCaseCommand) any {
	return struct {
		identityPayload
		SpeciesName    *string    `json:"speciesName,omitempty"`
		CollectionSite *string    `json:"collectionSite,omitempty"`
		CollectedAt    *time.Time `json:"collectedAt,omitempty"`
		ReceivedAt     *time.Time `json:"receivedAt,omitempty"`
		Owner          *string    `json:"owner,omitempty"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.SpeciesName, cmd.CollectionSite, cmd.CollectedAt, cmd.ReceivedAt, cmd.Owner}
}

func addLotsPayload(cmd AddLotsCommand) any {
	return struct {
		identityPayload
		Items []LotInput `json:"items"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.Items}
}

func generateSamplingPayload(ctx Context) any {
	return identityPayload{ctx.Actor, ctx.Role, ctx.ExpectedRevision}
}

func confirmSamplingPayload(cmd ConfirmSamplingCommand) any {
	return struct {
		identityPayload
		ActualCounts     map[string]int    `json:"actualCounts"`
		DeviationReasons map[string]string `json:"deviationReasons,omitempty"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.ActualCounts, cmd.DeviationReasons}
}

func recordTestPayload(cmd RecordTestCommand) any {
	return struct {
		identityPayload
		LotID             string          `json:"lotId"`
		TestType          domain.TestType `json:"testType"`
		ReplicateCounts   []int           `json:"replicateCounts"`
		TestedCount       int             `json:"testedCount"`
		PassedCount       int             `json:"passedCount"`
		ContaminatedCount int             `json:"contaminatedCount"`
		Threshold         float64         `json:"threshold"`
		IsRetest          bool            `json:"isRetest"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.LotID, cmd.TestType, cmd.ReplicateCounts, cmd.TestedCount, cmd.PassedCount, cmd.ContaminatedCount, cmd.Threshold, cmd.IsRetest}
}

func recordTestsPayload(cmd RecordTestsCommand) any {
	return struct {
		identityPayload
		Items []TestResultInput `json:"items"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, cmd.Items}
}

func assignFindingPayload(caseID, findingID string, cmd AssignFindingCommand) any {
	return struct {
		identityPayload
		CaseID    string    `json:"caseId"`
		FindingID string    `json:"findingId"`
		Assignee  string    `json:"assignee"`
		DueAt     time.Time `json:"dueAt"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, caseID, findingID, cmd.Assignee, cmd.DueAt}
}

func addEvidencePayload(caseID, findingID string, cmd AddEvidenceCommand) any {
	return struct {
		identityPayload
		CaseID      string `json:"caseId"`
		FindingID   string `json:"findingId"`
		Description string `json:"description"`
		Content     string `json:"content"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, caseID, findingID, cmd.Description, cmd.Content}
}

func remediatePayload(caseID, findingID string, cmd RemediateCommand) any {
	return struct {
		identityPayload
		CaseID    string `json:"caseId"`
		FindingID string `json:"findingId"`
		Evidence  string `json:"evidence,omitempty"`
		RetestID  string `json:"retestId"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, caseID, findingID, cmd.Evidence, cmd.RetestID}
}

func closeFindingPayload(caseID, findingID string, cmd CloseFindingCommand) any {
	return struct {
		identityPayload
		CaseID    string `json:"caseId"`
		FindingID string `json:"findingId"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, caseID, findingID}
}

func approvePayload(caseID string, cmd ApproveCommand) any {
	return struct {
		identityPayload
		CaseID string `json:"caseId"`
	}{identityPayload{cmd.Actor, cmd.Role, cmd.ExpectedRevision}, caseID}
}
