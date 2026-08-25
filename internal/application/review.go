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
	prepared, err := s.store.GetCase(caseID)
	if err != nil {
		return nil, mapStoreError(err)
	}
	if err := prepared.Freeze(cmd.Actor, cmd.Role, now); err != nil {
		return nil, err
	}
	cert := &domain.ReleaseCertificate{ID: s.id("certificate"), CaseID: prepared.ID, ManifestDigest: prepared.FrozenManifest.Digest, CaseRevision: prepared.Revision + 1, ApprovedBy: cmd.Actor, IssuedAt: now.UTC()}
	serial, err := s.store.AllocateCertificateRecord(cert, prepared.FrozenManifest)
	if err != nil {
		return nil, mapStoreError(err)
	}
	cert.SerialNumber = serial
	payload := repository.EventPayloadBuilder(func(_, after *domain.AccessionCase) any {
		return map[string]any{"approvedBy": cmd.Actor, "serialNumber": after.Certificate.SerialNumber, "manifestDigest": after.Certificate.ManifestDigest}
	})
	raw, _, err := s.store.Mutate(caseID, cmd.ExpectedRevision, cmd.IdempotencyKey, "certificate.issued", cmd.Actor, s.id("event"), now, payload, func(c *domain.AccessionCase, tx *bolt.Tx) (json.RawMessage, error) {
		if err := c.Freeze(cmd.Actor, cmd.Role, now); err != nil {
			return nil, err
		}
		if err := repository.AppendAuditEvent(tx, s.id("event"), c.ID, cmd.Actor, "case.frozen", c.Revision+1, now, map[string]any{"manifestDigest": c.FrozenManifest.Digest, "frozenBy": cmd.Actor}); err != nil {
			return nil, err
		}
		c.Certificate = cert
		c.Status = domain.StatusReleased
		return nil, nil
	})
	if err != nil {
		return nil, mapStoreError(err)
	}
	return decodeCase(raw)
}
