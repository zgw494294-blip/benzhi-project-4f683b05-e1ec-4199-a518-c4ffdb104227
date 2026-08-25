package application

import (
	"context"
	"encoding/json"
	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func (s *Service) CreateCase(cmd CreateCaseCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReceiver); err != nil {
		return nil, err
	}
	now := s.now()
	c, err := domain.NewCase(s.id("case"), cmd.AccessionCode, cmd.SpeciesName, cmd.CollectionSite, cmd.CollectedAt, cmd.ReceivedAt, cmd.Owner, now)
	if err != nil {
		return nil, err
	}
	c.Revision = 1
	raw, _ := json.Marshal(c)
	event, err := audit.NewEvent(s.id("event"), c.ID, cmd.Actor, "case.created", 1, 1, now, map[string]any{"accessionCode": c.AccessionCode, "speciesName": c.SpeciesName}, "")
	if err != nil {
		return nil, err
	}
	result, _, err := s.store.Create(repository.Mutation{Case: c, Event: event, IdempotencyKey: cmd.IdempotencyKey, Result: raw})
	if err != nil {
		return nil, mapStoreError(err)
	}
	return decodeCase(result)
}

func (s *Service) AddLot(caseID string, cmd AddLotCommand) (*domain.AccessionCase, error) {
	return s.AddLotContext(context.Background(), caseID, cmd)
}

func (s *Service) AddLotContext(requestContext context.Context, caseID string, cmd AddLotCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReceiver); err != nil {
		return nil, err
	}
	lot := domain.SeedLot{ID: s.id("lot"), LotCode: cmd.LotCode, ContainerCode: cmd.ContainerCode, SeedCount: cmd.SeedCount, MoisturePercent: cmd.MoisturePercent, ArrivalCondition: cmd.ArrivalCondition, ReservedCount: cmd.ReservedCount}
	return s.mutateContext(requestContext, caseID, cmd.Context, "lot.added", map[string]any{"lotCode": cmd.LotCode, "seedCount": cmd.SeedCount}, func(c *domain.AccessionCase) error { return c.AddLot(lot) })
}

func (s *Service) ReviseCase(caseID string, cmd ReviseCaseCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReceiver); err != nil {
		return nil, err
	}
	payload := repository.EventPayloadBuilder(func(before, after *domain.AccessionCase) any {
		summary := map[string]any{"changedFields": []string{}, "before": map[string]any{}, "after": map[string]any{}}
		beforeCopy := *before
		result, err := beforeCopy.ReviseBaseData(after.SpeciesName, after.CollectionSite, after.CollectedAt, after.ReceivedAt, after.Owner)
		if err == nil {
			return result
		}
		return summary
	})
	return s.mutate(caseID, cmd.Context, "case.base_data_revised", payload, func(c *domain.AccessionCase) error {
		species, site, collected, received, owner := c.SpeciesName, c.CollectionSite, c.CollectedAt, c.ReceivedAt, c.Owner
		if cmd.SpeciesName != nil {
			species = *cmd.SpeciesName
		}
		if cmd.CollectionSite != nil {
			site = *cmd.CollectionSite
		}
		if cmd.CollectedAt != nil {
			collected = *cmd.CollectedAt
		}
		if cmd.ReceivedAt != nil {
			received = *cmd.ReceivedAt
		}
		if cmd.Owner != nil {
			owner = *cmd.Owner
		}
		_, err := c.ReviseBaseData(species, site, collected, received, owner)
		return err
	})
}

func (s *Service) AddLots(caseID string, cmd AddLotsCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReceiver); err != nil {
		return nil, err
	}
	lots := make([]domain.SeedLot, len(cmd.Items))
	codes := make([]string, len(cmd.Items))
	for i, item := range cmd.Items {
		lots[i] = domain.SeedLot{ID: s.id("lot"), LotCode: item.LotCode, ContainerCode: item.ContainerCode, SeedCount: item.SeedCount, MoisturePercent: item.MoisturePercent, ArrivalCondition: item.ArrivalCondition, ReservedCount: item.ReservedCount}
		codes[i] = item.LotCode
	}
	payload := map[string]any{"successCount": len(lots), "lotCodes": codes}
	return s.mutate(caseID, cmd.Context, "lots.batch_added", payload, func(c *domain.AccessionCase) error { return c.AddLots(lots) })
}
