package application

import (
	"strings"

	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func (s *Service) GenerateSampling(caseID string, ctx Context) (*domain.AccessionCase, error) {
	if err := validateContext(ctx, domain.RoleReceiver); err != nil {
		return nil, err
	}
	return s.mutate(caseID, ctx, "sampling.generated", map[string]any{"ruleVersion": "SVR-2026.1"}, generateSamplingPayload(ctx), func(c *domain.AccessionCase) error { return c.GenerateSamplingPlan(s.id("sampling")) })
}

func (s *Service) ConfirmSampling(caseID string, cmd ConfirmSamplingCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReceiver); err != nil {
		return nil, err
	}
	deviations := make(map[string]string, len(cmd.DeviationReasons))
	for lotID, reason := range cmd.DeviationReasons {
		deviations[strings.TrimSpace(lotID)] = strings.TrimSpace(reason)
	}
	payload := repository.EventPayloadBuilder(func(_, after *domain.AccessionCase) any {
		count := 0
		reasons := map[string]string{}
		if after.Sampling != nil {
			for _, item := range after.Sampling.Items {
				if item.ActualCount != item.PlannedCount {
					count++
					reasons[item.LotCode] = domain.DigestText(item.DeviationReason)
				}
			}
		}
		return map[string]any{"actualCounts": cmd.ActualCounts, "deviationCount": count, "deviationReasonDigests": reasons}
	})
	return s.mutate(caseID, cmd.Context, "sampling.confirmed", payload, confirmSamplingPayload(cmd), func(c *domain.AccessionCase) error {
		return c.ConfirmSamplingWithDeviations(cmd.ActualCounts, deviations, cmd.Actor, s.now())
	})
}
