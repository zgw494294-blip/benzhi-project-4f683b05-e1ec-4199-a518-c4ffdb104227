package application

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/domain"
	"seed-vault-release/internal/repository"
)

type Service struct {
	store *repository.Store
	now   func() time.Time
	id    func(string) string
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

func (s *Service) mutate(caseID string, ctx Context, action string, payload any, requestPayload any, fn func(*domain.AccessionCase) error) (*domain.AccessionCase, error) {
	raw, _, err := s.store.Mutate(caseID, ctx.ExpectedRevision, ctx.IdempotencyKey, action, ctx.Actor, s.id("event"), s.now(), payload, requestDigest(caseID, action, requestPayload), func(c *domain.AccessionCase, _ *bolt.Tx) (json.RawMessage, error) {
		return nil, fn(c)
	})
	if err != nil {
		return nil, mapStoreError(err)
	}
	return decodeCase(raw)
}

// requestDigest 计算请求指纹摘要：仅当目标、调用身份与业务载荷完全一致的重试才视为同一请求。
func requestDigest(caseID, action string, requestPayload any) string {
	body := map[string]any{
		"action":        action,
		"caseId":        caseID,
		"requestPayload": requestPayload,
	}
	b, err := json.Marshal(body)
	if err != nil {
		return ""
	}
	return domain.DigestText(string(b))
}
