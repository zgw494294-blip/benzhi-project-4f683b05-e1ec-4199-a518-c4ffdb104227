package application

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

type Service struct {
	store           *repository.Store
	now             func() time.Time
	id              func(string) string
	timelineCacheMu sync.Mutex
	timelineCache   cachedTimeline
}

func New(store *repository.Store) *Service {
	return &Service{store: store, now: time.Now, id: newID}
}

func newID(prefix string) string {
	b := make([]byte, 12)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
	}
	return prefix + "-" + hex.EncodeToString(b)
}

func validateContext(ctx Context, roles ...domain.Role) error {
	if err := domain.RequireText("操作者", ctx.Actor); err != nil {
		return err
	}
	if err := domain.RequireText("idempotencyKey", ctx.IdempotencyKey); err != nil {
		return err
	}
	if len(ctx.IdempotencyKey) > 128 {
		return domain.Invalid("idempotencyKey过长")
	}
	return domain.ValidateRole(ctx.Role, roles...)
}

func decodeCase(raw json.RawMessage) (*domain.AccessionCase, error) {
	var c domain.AccessionCase
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

func mapStoreError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrRevision) {
		return domain.Conflict("expectedRevision与当前案卷版本不一致")
	}
	if errors.Is(err, repository.ErrNotFound) {
		return domain.NotFound("案卷不存在")
	}
	if errors.Is(err, repository.ErrDuplicate) {
		return domain.Conflict("记录已存在")
	}
	return err
}

func (s *Service) mutate(caseID string, ctx Context, action string, payload any, fn func(*domain.AccessionCase) error) (*domain.AccessionCase, error) {
	raw, _, err := s.store.MutateCase(caseID, ctx.ExpectedRevision, ctx.IdempotencyKey, action, ctx.Actor, s.id("event"), s.now(), payload, fn)
	if err != nil {
		return nil, mapStoreError(err)
	}
	s.invalidateTimeline(caseID)
	return decodeCase(raw)
}
