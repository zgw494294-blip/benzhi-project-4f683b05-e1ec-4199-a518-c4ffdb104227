package application

import "seed-vault-release/internal/domain"

func (s *Service) RemediateFinding(caseID, findingID string, cmd RemediateCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	payload := map[string]any{"findingId": findingID, "retestId": cmd.RetestID}
	return s.mutate(caseID, cmd.Context, "finding.remediated", payload, remediatePayload(caseID, findingID, cmd), func(c *domain.AccessionCase) error {
		return c.SubmitFindingRemediation(findingID, cmd.Actor, cmd.RetestID)
	})
}

func (s *Service) AssignFinding(caseID, findingID string, cmd AssignFindingCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	payload := map[string]any{"findingId": findingID, "assignee": cmd.Assignee, "dueAt": cmd.DueAt.UTC()}
	return s.mutate(caseID, cmd.Context, "finding.assigned", payload, assignFindingPayload(caseID, findingID, cmd), func(c *domain.AccessionCase) error { return c.AssignFinding(findingID, cmd.Assignee, cmd.DueAt) })
}

func (s *Service) AddFindingEvidence(caseID, findingID string, cmd AddEvidenceCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	if err := domain.RequireText("证据内容", cmd.Content); err != nil {
		return nil, err
	}
	now := s.now()
	evidence := domain.FindingEvidence{ID: s.id("evidence"), SubmittedBy: cmd.Actor, Description: cmd.Description, ContentDigest: domain.DigestText(cmd.Content), SubmittedAt: now}
	payload := map[string]any{"findingId": findingID, "description": cmd.Description, "contentDigest": evidence.ContentDigest}
	return s.mutate(caseID, cmd.Context, "finding.evidence_added", payload, addEvidencePayload(caseID, findingID, cmd), func(c *domain.AccessionCase) error { return c.AddFindingEvidence(findingID, evidence) })
}

func (s *Service) CloseFinding(caseID, findingID string, cmd CloseFindingCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleTester); err != nil {
		return nil, err
	}
	return s.mutate(caseID, cmd.Context, "finding.closed", map[string]any{"findingId": findingID}, closeFindingPayload(caseID, findingID, cmd), func(c *domain.AccessionCase) error { return c.CloseFinding(findingID, cmd.Actor, cmd.Role, s.now()) })
}
