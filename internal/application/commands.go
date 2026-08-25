package application

import (
	"seed-vault-release/internal/domain"
	"time"
)

type Context struct {
	Actor            string      `json:"actor"`
	Role             domain.Role `json:"role"`
	ExpectedRevision uint64      `json:"expectedRevision"`
	IdempotencyKey   string      `json:"idempotencyKey"`
}

type CreateCaseCommand struct {
	Context
	AccessionCode  string    `json:"accessionCode"`
	SpeciesName    string    `json:"speciesName"`
	CollectionSite string    `json:"collectionSite"`
	CollectedAt    time.Time `json:"collectedAt"`
	ReceivedAt     time.Time `json:"receivedAt"`
	Owner          string    `json:"owner"`
}

type AddLotCommand struct {
	Context
	LotCode          string  `json:"lotCode"`
	ContainerCode    string  `json:"containerCode"`
	SeedCount        int     `json:"seedCount"`
	MoisturePercent  float64 `json:"moisturePercent"`
	ArrivalCondition string  `json:"arrivalCondition"`
	ReservedCount    int     `json:"reservedCount"`
}

type ReviseCaseCommand struct {
	Context
	SpeciesName    *string    `json:"speciesName,omitempty"`
	CollectionSite *string    `json:"collectionSite,omitempty"`
	CollectedAt    *time.Time `json:"collectedAt,omitempty"`
	ReceivedAt     *time.Time `json:"receivedAt,omitempty"`
	Owner          *string    `json:"owner,omitempty"`
}

type LotInput struct {
	LotCode          string  `json:"lotCode"`
	ContainerCode    string  `json:"containerCode"`
	SeedCount        int     `json:"seedCount"`
	MoisturePercent  float64 `json:"moisturePercent"`
	ArrivalCondition string  `json:"arrivalCondition"`
	ReservedCount    int     `json:"reservedCount"`
}

type AddLotsCommand struct {
	Context
	Items []LotInput `json:"items"`
}

type ConfirmSamplingCommand struct {
	Context
	ActualCounts     map[string]int    `json:"actualCounts"`
	DeviationReasons map[string]string `json:"deviationReasons,omitempty"`
}

type TestResultInput struct {
	LotID             string          `json:"lotId"`
	TestType          domain.TestType `json:"testType"`
	ReplicateCounts   []int           `json:"replicateCounts"`
	TestedCount       int             `json:"testedCount"`
	PassedCount       int             `json:"passedCount"`
	ContaminatedCount int             `json:"contaminatedCount"`
	Threshold         float64         `json:"threshold"`
}

type RecordTestsCommand struct {
	Context
	Items []TestResultInput `json:"items"`
}

type RecordTestCommand struct {
	Context
	LotID             string          `json:"lotId"`
	TestType          domain.TestType `json:"testType"`
	ReplicateCounts   []int           `json:"replicateCounts"`
	TestedCount       int             `json:"testedCount"`
	PassedCount       int             `json:"passedCount"`
	ContaminatedCount int             `json:"contaminatedCount"`
	Threshold         float64         `json:"threshold"`
	IsRetest          bool            `json:"isRetest"`
}

type RemediateCommand struct {
	Context
	Evidence string `json:"evidence,omitempty"`
	RetestID string `json:"retestId"`
}

type AssignFindingCommand struct {
	Context
	Assignee string    `json:"assignee"`
	DueAt    time.Time `json:"dueAt"`
}

type AddEvidenceCommand struct {
	Context
	Description string `json:"description"`
	Content     string `json:"content"`
}

type CloseFindingCommand struct{ Context }
type ApproveCommand struct{ Context }
