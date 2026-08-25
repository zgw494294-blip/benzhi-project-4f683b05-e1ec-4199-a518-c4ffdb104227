package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
)

type Mutation struct {
	Case           *domain.AccessionCase
	Event          audit.Event
	IdempotencyKey string
	Result         json.RawMessage
}

type IdempotencyRecord struct {
	CaseID       string          `json:"caseId"`
	Action       string          `json:"action"`
	Result       json.RawMessage `json:"result"`
	ErrorCode    string          `json:"errorCode,omitempty"`
	ErrorMessage string          `json:"errorMessage,omitempty"`
	CreatedAt    time.Time       `json:"createdAt"`
}

type EventPayloadBuilder func(before, after *domain.AccessionCase) any

func (s *Store) Create(m Mutation) (json.RawMessage, bool, error) {
	var result json.RawMessage
	var replay bool
	err := s.db.Update(func(tx *bolt.Tx) error {
		if cached := tx.Bucket(bucketIdempotency).Get([]byte(m.IdempotencyKey)); cached != nil {
			var rec IdempotencyRecord
			if err := json.Unmarshal(cached, &rec); err != nil {
				return fmt.Errorf("损坏的幂等记录: %w", err)
			}
			result, replay = cloneBytes(rec.Result), true
			return nil
		}
		cases := tx.Bucket(bucketCases)
		if cases.Get([]byte(m.Case.ID)) != nil {
			return ErrDuplicate
		}
		if tx.Bucket(bucketAccessions).Get([]byte(strings.ToLower(m.Case.AccessionCode))) != nil {
			return domain.Conflict("案卷编号已存在")
		}
		m.Case.Revision = 1
		m.Event.CaseRevision = 1
		m.Event.Digest = m.Event.ComputeDigest()
		if err := putAggregate(tx, nil, m.Case); err != nil {
			return err
		}
		if err := putEvent(tx, m.Event); err != nil {
			return err
		}
		if err := putIdempotency(tx, m.IdempotencyKey, m.Case.ID, m.Event.Action, m.Result); err != nil {
			return err
		}
		result = cloneBytes(m.Result)
		return nil
	})
	return result, replay, err
}

func (s *Store) Mutate(caseID string, expected uint64, key, action, actor, eventID string, now time.Time, payload any, fn func(*domain.AccessionCase, *bolt.Tx) (json.RawMessage, error)) (json.RawMessage, bool, error) {
	var result json.RawMessage
	var replay bool
	var committedError error
	err := s.db.Update(func(tx *bolt.Tx) error {
		if cached := tx.Bucket(bucketIdempotency).Get([]byte(key)); cached != nil {
			var rec IdempotencyRecord
			if err := json.Unmarshal(cached, &rec); err != nil {
				return err
			}
			if rec.CaseID != caseID || rec.Action != action {
				return domain.Conflict("idempotencyKey已用于其他操作")
			}
			if rec.ErrorCode != "" {
				committedError = &domain.BusinessError{Code: domain.ErrorCode(rec.ErrorCode), Message: rec.ErrorMessage}
				replay = true
				return nil
			}
			result, replay = cloneBytes(rec.Result), true
			return nil
		}
		current, err := loadCase(tx, caseID)
		if err != nil {
			return err
		}
		if current.Revision != expected {
			return ErrRevision
		}
		beforeStatus := current.Status
		beforeBytes, err := json.Marshal(current)
		if err != nil {
			return err
		}
		var before domain.AccessionCase
		if err := json.Unmarshal(beforeBytes, &before); err != nil {
			return err
		}
		result, err = fn(current, tx)
		if err != nil {
			var business *domain.BusinessError
			if errors.As(err, &business) {
				if putErr := putIdempotencyFailure(tx, key, caseID, action, business); putErr != nil {
					return putErr
				}
				committedError = business
				return nil
			}
			return err
		}
		current.Revision++
		current.UpdatedAt = now.UTC()
		if len(result) == 0 {
			result, err = json.Marshal(current)
			if err != nil {
				return err
			}
		}
		events, err := loadTimeline(tx, caseID)
		if err != nil {
			return err
		}
		eventPayload := payload
		if builder, ok := payload.(EventPayloadBuilder); ok {
			eventPayload = builder(&before, current)
		}
		event, err := audit.NewEvent(eventID, caseID, actor, action, current.Revision, uint64(len(events)+1), now, eventPayload, audit.LastDigest(events))
		if err != nil {
			return err
		}
		if err := putAggregate(tx, &beforeStatus, current); err != nil {
			return err
		}
		if err := putEvent(tx, event); err != nil {
			return err
		}
		if err := putIdempotency(tx, key, caseID, action, result); err != nil {
			return err
		}
		return nil
	})
	if err == nil && committedError != nil {
		return nil, replay, committedError
	}
	return result, replay, err
}

func (s *Store) MutateCase(caseID string, expected uint64, key, action, actor, eventID string, now time.Time, payload any, fn func(*domain.AccessionCase) error) (json.RawMessage, bool, error) {
	return s.Mutate(caseID, expected, key, action, actor, eventID, now, payload, func(c *domain.AccessionCase, _ *bolt.Tx) (json.RawMessage, error) {
		return nil, fn(c)
	})
}

func (s *Store) MutateCaseContext(ctx context.Context, caseID string, expected uint64, key, action, actor, eventID string, now time.Time, payload any, fn func(*domain.AccessionCase) error) (json.RawMessage, bool, error) {
	if err := ctx.Err(); err != nil {
		return nil, false, err
	}
	return s.MutateCase(caseID, expected, key, action, actor, eventID, now, payload, fn)
}

func putAggregate(tx *bolt.Tx, before *domain.Status, c *domain.AccessionCase) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}
	if err := tx.Bucket(bucketCases).Put([]byte(c.ID), b); err != nil {
		return err
	}
	if before != nil {
		_ = tx.Bucket(bucketStatus).Delete(statusKey(string(*before), c.ID))
	}
	if err := tx.Bucket(bucketStatus).Put(statusKey(string(c.Status), c.ID), []byte(c.ID)); err != nil {
		return err
	}
	if err := tx.Bucket(bucketAccessions).Put([]byte(strings.ToLower(c.AccessionCode)), []byte(c.ID)); err != nil {
		return err
	}
	return nil
}

func putEvent(tx *bolt.Tx, event audit.Event) error {
	b, err := json.Marshal(event)
	if err != nil {
		return err
	}
	key := auditKey(event.CaseID, event.Sequence)
	if tx.Bucket(bucketAudit).Get(key) != nil {
		return ErrImmutable
	}
	return tx.Bucket(bucketAudit).Put(key, b)
}

func putIdempotency(tx *bolt.Tx, key, caseID, action string, result []byte) error {
	if strings.TrimSpace(key) == "" {
		return domain.Invalid("idempotencyKey不能为空")
	}
	rec, _ := json.Marshal(IdempotencyRecord{CaseID: caseID, Action: action, Result: result, CreatedAt: time.Now().UTC()})
	return tx.Bucket(bucketIdempotency).Put([]byte(key), rec)
}

func putIdempotencyFailure(tx *bolt.Tx, key, caseID, action string, business *domain.BusinessError) error {
	if strings.TrimSpace(key) == "" {
		return domain.Invalid("idempotencyKey不能为空")
	}
	rec, _ := json.Marshal(IdempotencyRecord{CaseID: caseID, Action: action, ErrorCode: string(business.Code), ErrorMessage: business.Message, CreatedAt: time.Now().UTC()})
	return tx.Bucket(bucketIdempotency).Put([]byte(key), rec)
}

func AppendAuditEvent(tx *bolt.Tx, id, caseID, actor, action string, revision uint64, at time.Time, payload any) error {
	events, err := loadTimeline(tx, caseID)
	if err != nil {
		return err
	}
	event, err := audit.NewEvent(id, caseID, actor, action, revision, uint64(len(events)+1), at, payload, audit.LastDigest(events))
	if err != nil {
		return err
	}
	return putEvent(tx, event)
}

func AllocateCertificate(tx *bolt.Tx, cert *domain.ReleaseCertificate, manifest *domain.FrozenManifest) (uint64, error) {
	b := tx.Bucket(bucketCertificates)
	serial, err := b.NextSequence()
	if err != nil {
		return 0, err
	}
	cert.SerialNumber = serial
	key := certificateKey(serial)
	if b.Get(key) != nil {
		return 0, ErrImmutable
	}
	certBytes, _ := json.Marshal(cert)
	manifestBytes, _ := json.Marshal(manifest)
	if err := b.Put(key, certBytes); err != nil {
		return 0, err
	}
	manifestKey := []byte(strconv.FormatUint(serial, 10))
	if tx.Bucket(bucketManifests).Get(manifestKey) != nil {
		return 0, ErrImmutable
	}
	if err := tx.Bucket(bucketManifests).Put(manifestKey, manifestBytes); err != nil {
		return 0, err
	}
	return serial, nil
}
