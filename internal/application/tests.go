package application

import (
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func (s *Service) RecordTest(caseID string, cmd RecordTestCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	in := domain.TestInput{ID: s.id("test"), LotID: cmd.LotID, Type: cmd.TestType, Replicates: cmd.ReplicateCounts, Tested: cmd.TestedCount, Passed: cmd.PassedCount, Contaminated: cmd.ContaminatedCount, Threshold: cmd.Threshold, Actor: cmd.Actor, At: s.now(), IsRetest: cmd.IsRetest}
	payload := map[string]any{"lotId": cmd.LotID, "testType": cmd.TestType, "testedCount": cmd.TestedCount, "isRetest": cmd.IsRetest}
	return s.mutate(caseID, cmd.Context, "test.recorded", payload, recordTestPayload(cmd), func(c *domain.AccessionCase) error { _, _, err := c.RecordTest(in); return err })
}

func (s *Service) RecordTests(caseID string, cmd RecordTestsCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	now := s.now()
	inputs := make([]domain.TestInput, len(cmd.Items))
	lotIDs := make([]string, 0, len(cmd.Items))
	abnormal := 0
	for i, item := range cmd.Items {
		inputs[i] = domain.TestInput{ID: s.id("test"), LotID: item.LotID, Type: item.TestType, Replicates: item.ReplicateCounts, Tested: item.TestedCount, Passed: item.PassedCount, Contaminated: item.ContaminatedCount, Threshold: item.Threshold, Actor: cmd.Actor, At: now}
		assessment, err := domain.AssessTest(domain.ViabilityTest{ID: inputs[i].ID, TestType: item.TestType, TestedCount: item.TestedCount, PassedCount: item.PassedCount, ContaminatedCount: item.ContaminatedCount, Threshold: item.Threshold})
		if err == nil && assessment.Result == domain.ResultFail {
			abnormal++
		}
		lotIDs = append(lotIDs, item.LotID)
	}
	payload := repository.EventPayloadBuilder(func(_, after *domain.AccessionCase) any {
		codes := make([]string, 0, len(lotIDs))
		seen := map[string]bool{}
		for _, lotID := range lotIDs {
			if seen[lotID] {
				continue
			}
			if lot, err := after.Lot(lotID); err == nil {
				codes = append(codes, lot.LotCode)
				seen[lotID] = true
			}
		}
		return map[string]any{"resultCount": len(inputs), "abnormalCount": abnormal, "affectedLotCodes": codes}
	})
	return s.mutate(caseID, cmd.Context, "tests.batch_recorded", payload, recordTestsPayload(cmd), func(c *domain.AccessionCase) error { _, _, err := c.RecordTests(inputs); return err })
}
