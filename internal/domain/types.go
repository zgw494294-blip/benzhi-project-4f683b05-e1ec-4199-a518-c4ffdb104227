package domain

import "time"

type Status string

const (
	StatusDraft       Status = "草稿"
	StatusSampling    Status = "待抽样"
	StatusTesting     Status = "检验中"
	StatusRemediation Status = "整改中"
	StatusReview      Status = "待审核"
	StatusReleased    Status = "已放行"
)

type Role string

const (
	RoleReceiver Role = "receiver"
	RoleTester   Role = "tester"
	RoleReviewer Role = "reviewer"
)

type AccessionCase struct {
	ID             string              `json:"id"`
	AccessionCode  string              `json:"accessionCode"`
	SpeciesName    string              `json:"speciesName"`
	CollectionSite string              `json:"collectionSite"`
	CollectedAt    time.Time           `json:"collectedAt"`
	ReceivedAt     time.Time           `json:"receivedAt"`
	Owner          string              `json:"owner"`
	Status         Status              `json:"status"`
	Revision       uint64              `json:"revision"`
	CreatedAt      time.Time           `json:"createdAt"`
	UpdatedAt      time.Time           `json:"updatedAt"`
	Lots           []SeedLot           `json:"lots"`
	Sampling       *SamplingPlan       `json:"sampling,omitempty"`
	Tests          []ViabilityTest     `json:"tests"`
	Findings       []Finding           `json:"findings"`
	FrozenManifest *FrozenManifest     `json:"frozenManifest,omitempty"`
	Certificate    *ReleaseCertificate `json:"certificate,omitempty"`
}

type SeedLot struct {
	ID               string  `json:"id"`
	CaseID           string  `json:"caseId"`
	LotCode          string  `json:"lotCode"`
	ContainerCode    string  `json:"containerCode"`
	SeedCount        int     `json:"seedCount"`
	MoisturePercent  float64 `json:"moisturePercent"`
	ArrivalCondition string  `json:"arrivalCondition"`
	ReservedCount    int     `json:"reservedCount"`
}

type SamplingItem struct {
	LotID           string `json:"lotId"`
	LotCode         string `json:"lotCode"`
	PlannedCount    int    `json:"plannedCount"`
	ActualCount     int    `json:"actualCount"`
	ReservedCount   int    `json:"reservedCount"`
	AvailableCount  int    `json:"availableCount"`
	DeviationReason string `json:"deviationReason,omitempty"`
}

type SamplingPlan struct {
	ID          string         `json:"id"`
	CaseID      string         `json:"caseId"`
	RuleVersion string         `json:"ruleVersion"`
	Items       []SamplingItem `json:"items"`
	ConfirmedBy string         `json:"confirmedBy,omitempty"`
	ConfirmedAt *time.Time     `json:"confirmedAt,omitempty"`
}

type TestType string

const (
	TestGermination   TestType = "germination"
	TestContamination TestType = "contamination"
)

type TestResult string

const (
	ResultPass TestResult = "pass"
	ResultFail TestResult = "fail"
)

type ViabilityTest struct {
	ID                string     `json:"id"`
	CaseID            string     `json:"caseId"`
	LotID             string     `json:"lotId"`
	TestType          TestType   `json:"testType"`
	ReplicateCounts   []int      `json:"replicateCounts"`
	TestedCount       int        `json:"testedCount"`
	PassedCount       int        `json:"passedCount"`
	ContaminatedCount int        `json:"contaminatedCount"`
	Threshold         float64    `json:"threshold"`
	Result            TestResult `json:"result"`
	PerformedBy       string     `json:"performedBy"`
	PerformedAt       time.Time  `json:"performedAt"`
	IsRetest          bool       `json:"isRetest"`
}

type FindingStatus string

const (
	FindingOpen       FindingStatus = "open"
	FindingRemediated FindingStatus = "remediated"
	FindingClosed     FindingStatus = "closed"
)

type Finding struct {
	ID                  string            `json:"id"`
	CaseID              string            `json:"caseId"`
	LotID               string            `json:"lotId"`
	SourceTestID        string            `json:"sourceTestId"`
	Category            string            `json:"category"`
	Severity            string            `json:"severity"`
	Status              FindingStatus     `json:"status"`
	CreatedAt           time.Time         `json:"createdAt"`
	Assignee            string            `json:"assignee,omitempty"`
	DueAt               *time.Time        `json:"dueAt,omitempty"`
	EvidenceLedger      []FindingEvidence `json:"evidenceLedger"`
	Timeliness          string            `json:"timeliness,omitempty"`
	RemediationEvidence string            `json:"remediationEvidence,omitempty"`
	RemediatedBy        string            `json:"remediatedBy,omitempty"`
	RetestID            string            `json:"retestId,omitempty"`
	ClosedBy            string            `json:"closedBy,omitempty"`
	ClosedAt            *time.Time        `json:"closedAt,omitempty"`
}

type FindingEvidence struct {
	ID            string    `json:"id"`
	SubmittedBy   string    `json:"submittedBy"`
	Description   string    `json:"description"`
	ContentDigest string    `json:"contentDigest"`
	SubmittedAt   time.Time `json:"submittedAt"`
}

type FrozenManifest struct {
	CanonicalJSON string    `json:"canonicalJson"`
	Digest        string    `json:"digest"`
	FrozenAt      time.Time `json:"frozenAt"`
	FrozenBy      string    `json:"frozenBy"`
}

type ReleaseCertificate struct {
	ID             string    `json:"id"`
	CaseID         string    `json:"caseId"`
	SerialNumber   uint64    `json:"serialNumber"`
	ManifestDigest string    `json:"manifestDigest"`
	CaseRevision   uint64    `json:"caseRevision"`
	ApprovedBy     string    `json:"approvedBy"`
	IssuedAt       time.Time `json:"issuedAt"`
}
