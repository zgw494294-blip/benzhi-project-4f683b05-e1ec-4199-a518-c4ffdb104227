package application

import (
	"encoding/json"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

func (s *Service) Approve(caseID string, cmd ApproveCommand) (*domain.AccessionCase, error) {
	if err := validateContext(cmd.Context, domain.RoleReviewer); err != nil {
		return nil, err
	}
	now := s.now()
	payload := repository.EventPayloadBuilder(func(_, after *domain.AccessionCase) any {
		return map[string]any{"approvedBy": cmd.Actor, "serialNumber": after.Certificate.SerialNumber, "manifestDigest": after.Certificate.ManifestDigest}
	})
	raw, _, err := s.store.Mutate(caseID, cmd.ExpectedRevision, cmd.IdempotencyKey, "certificate.issued", cmd.Actor, s.id("event"), now, payload, requestDigest(caseID, "certificate.issued", approvePayload(caseID, cmd)), func(c *domain.AccessionCase, tx *bolt.Tx) (json.RawMessage, error) {
		if err := c.Freeze(cmd.Actor, cmd.Role, now); err != nil {
			return nil, err
		}
		if err := repository.AppendAuditEvent(tx, s.id("event"), c.ID, cmd.Actor, "case.frozen", c.Revision+1, now, map[string]any{"manifestDigest": c.FrozenManifest.Digest, "frozenBy": cmd.Actor}); err != nil {
			return nil, err
		}
		cert := &domain.ReleaseCertificate{ID: s.id("certificate"), CaseID: c.ID, ManifestDigest: c.FrozenManifest.Digest, CaseRevision: c.Revision + 1, ApprovedBy: cmd.Actor, IssuedAt: now.UTC()}
		serial, err := repository.AllocateCertificate(tx, cert, c.FrozenManifest)
		if err != nil {
			return nil, err
		}
		cert.SerialNumber = serial
		c.Certificate = cert
		c.Status = domain.StatusReleased
		return nil, nil
	})
	if err != nil {
		return nil, mapStoreError(err)
	}
	return decodeCase(raw)
}
