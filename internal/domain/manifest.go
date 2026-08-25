package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"time"
)

type manifestLot struct {
	LotCode         string  `json:"lotCode"`
	ContainerCode   string  `json:"containerCode"`
	SeedCount       int     `json:"seedCount"`
	ReservedCount   int     `json:"reservedCount"`
	MoisturePercent float64 `json:"moisturePercent"`
}

type manifestSampling struct {
	RuleVersion     string `json:"ruleVersion"`
	LotCode         string `json:"lotCode"`
	PlannedCount    int    `json:"plannedCount"`
	ActualCount     int    `json:"actualCount"`
	ReservedCount   int    `json:"reservedCount"`
	AvailableCount  int    `json:"availableCount"`
	DeviationReason string `json:"deviationReason,omitempty"`
}

type manifestTest struct {
	ID                string     `json:"id"`
	LotCode           string     `json:"lotCode"`
	Type              TestType   `json:"type"`
	TestedCount       int        `json:"testedCount"`
	PassedCount       int        `json:"passedCount"`
	ContaminatedCount int        `json:"contaminatedCount"`
	Threshold         float64    `json:"threshold"`
	Result            TestResult `json:"result"`
	IsRetest          bool       `json:"isRetest"`
}

type manifestFinding struct {
	ID             string `json:"id"`
	LotCode        string `json:"lotCode"`
	Category       string `json:"category"`
	EvidenceDigest string `json:"evidenceDigest"`
	RetestID       string `json:"retestId"`
	ClosedBy       string `json:"closedBy"`
}

type manifestPayload struct {
	AccessionCode  string             `json:"accessionCode"`
	SpeciesName    string             `json:"speciesName"`
	CollectionSite string             `json:"collectionSite"`
	CollectedAt    time.Time          `json:"collectedAt"`
	ReceivedAt     time.Time          `json:"receivedAt"`
	Owner          string             `json:"owner"`
	Lots           []manifestLot      `json:"lots"`
	Sampling       []manifestSampling `json:"sampling"`
	Tests          []manifestTest     `json:"tests"`
	ClosedFindings []manifestFinding  `json:"closedFindings"`
}

func BuildManifest(c *AccessionCase) (string, string, error) {
	lotCodes := make(map[string]string, len(c.Lots))
	lots := make([]manifestLot, 0, len(c.Lots))
	for _, lot := range c.Lots {
		lotCodes[lot.ID] = lot.LotCode
		lots = append(lots, manifestLot{LotCode: lot.LotCode, ContainerCode: lot.ContainerCode, SeedCount: lot.SeedCount, ReservedCount: lot.ReservedCount, MoisturePercent: lot.MoisturePercent})
	}
	sort.Slice(lots, func(i, j int) bool { return lots[i].LotCode < lots[j].LotCode })
	sampling := []manifestSampling{}
	if c.Sampling != nil {
		for _, item := range c.Sampling.Items {
			sampling = append(sampling, manifestSampling{RuleVersion: c.Sampling.RuleVersion, LotCode: lotCodes[item.LotID], PlannedCount: item.PlannedCount, ActualCount: item.ActualCount, ReservedCount: item.ReservedCount, AvailableCount: item.AvailableCount, DeviationReason: item.DeviationReason})
		}
	}
	sort.Slice(sampling, func(i, j int) bool { return sampling[i].LotCode < sampling[j].LotCode })
	tests := make([]manifestTest, 0, len(c.Tests))
	for _, test := range c.Tests {
		tests = append(tests, manifestTest{ID: test.ID, LotCode: lotCodes[test.LotID], Type: test.TestType, TestedCount: test.TestedCount, PassedCount: test.PassedCount, ContaminatedCount: test.ContaminatedCount, Threshold: test.Threshold, Result: test.Result, IsRetest: test.IsRetest})
	}
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].LotCode != tests[j].LotCode {
			return tests[i].LotCode < tests[j].LotCode
		}
		if tests[i].Type != tests[j].Type {
			return tests[i].Type < tests[j].Type
		}
		return tests[i].ID < tests[j].ID
	})
	findings := make([]manifestFinding, 0, len(c.Findings))
	for _, finding := range c.Findings {
		evidenceDigest := DigestText(finding.RemediationEvidence)
		if len(finding.EvidenceLedger) > 0 {
			evidenceDigest = DigestEvidenceLedger(finding.EvidenceLedger)
		}
		findings = append(findings, manifestFinding{ID: finding.ID, LotCode: lotCodes[finding.LotID], Category: finding.Category, EvidenceDigest: evidenceDigest, RetestID: finding.RetestID, ClosedBy: finding.ClosedBy})
	}
	sort.Slice(findings, func(i, j int) bool { return findings[i].ID < findings[j].ID })
	payload := manifestPayload{AccessionCode: c.AccessionCode, SpeciesName: c.SpeciesName, CollectionSite: c.CollectionSite, CollectedAt: c.CollectedAt, ReceivedAt: c.ReceivedAt, Owner: c.Owner, Lots: lots, Sampling: sampling, Tests: tests, ClosedFindings: findings}
	b, err := json.Marshal(payload)
	if err != nil {
		return "", "", err
	}
	sum := sha256.Sum256(b)
	return string(b), hex.EncodeToString(sum[:]), nil
}

func ManifestBusinessSummary(canonical string) (json.RawMessage, error) {
	var payload manifestPayload
	if err := json.Unmarshal([]byte(canonical), &payload); err != nil {
		return nil, Invalid("冻结清单内容无法解析")
	}
	b, err := json.Marshal(payload)
	return json.RawMessage(b), err
}

func (c *AccessionCase) Freeze(actor string, role Role, now time.Time) error {
	if err := ValidateRole(role, RoleReviewer); err != nil {
		return err
	}
	if c.Status != StatusReview || !c.ReadyForReview() {
		return Conflict("案卷资料尚不完整，不能审核冻结")
	}
	if actor == c.Owner {
		return Forbidden("案卷负责人不能审核自己的案卷")
	}
	canonical, digest, err := BuildManifest(c)
	if err != nil {
		return err
	}
	c.FrozenManifest = &FrozenManifest{CanonicalJSON: canonical, Digest: digest, FrozenAt: now.UTC(), FrozenBy: actor}
	return nil
}

func DigestText(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
