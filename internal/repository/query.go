package repository

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"
	"strings"

	bolt "go.etcd.io/bbolt"
	"seed-vault-release/internal/audit"
	"seed-vault-release/internal/domain"
)

func loadCase(tx *bolt.Tx, id string) (*domain.AccessionCase, error) {
	b := tx.Bucket(bucketCases).Get([]byte(id))
	if b == nil {
		return nil, ErrNotFound
	}
	var c domain.AccessionCase
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, &domain.BusinessError{Code: domain.CodeCorrupt, Message: "案卷记录损坏"}
	}
	if c.ID != id || c.Revision == 0 {
		return nil, &domain.BusinessError{Code: domain.CodeCorrupt, Message: "案卷记录标识或版本损坏"}
	}
	return &c, nil
}

func (s *Store) GetCase(id string) (*domain.AccessionCase, error) {
	s.caseCacheMu.RLock()
	cached := cloneBytes(s.caseCache[id])
	s.caseCacheMu.RUnlock()
	if cached != nil {
		var c domain.AccessionCase
		if err := json.Unmarshal(cached, &c); err != nil {
			return nil, &domain.BusinessError{Code: domain.CodeCorrupt, Message: "案卷缓存损坏"}
		}
		return &c, nil
	}
	var c *domain.AccessionCase
	err := s.db.View(func(tx *bolt.Tx) error { var err error; c, err = loadCase(tx, id); return err })
	if err == nil {
		encoded, marshalErr := json.Marshal(c)
		if marshalErr != nil {
			return nil, marshalErr
		}
		s.caseCacheMu.Lock()
		s.caseCache[id] = cloneBytes(encoded)
		s.caseCacheMu.Unlock()
	}
	return c, err
}

func (s *Store) CaseByAccession(code string) (*domain.AccessionCase, error) {
	var c *domain.AccessionCase
	err := s.db.View(func(tx *bolt.Tx) error {
		id := tx.Bucket(bucketAccessions).Get([]byte(strings.ToLower(strings.TrimSpace(code))))
		if id == nil {
			return ErrNotFound
		}
		var err error
		c, err = loadCase(tx, string(id))
		return err
	})
	return c, err
}

func (s *Store) ListCases(status domain.Status) ([]domain.AccessionCase, error) {
	result := []domain.AccessionCase{}
	err := s.db.View(func(tx *bolt.Tx) error {
		if status != "" {
			prefix := []byte(string(status) + "\x00")
			cur := tx.Bucket(bucketStatus).Cursor()
			for k, v := cur.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cur.Next() {
				c, err := loadCase(tx, string(v))
				if err != nil {
					return err
				}
				result = append(result, *c)
			}
			return nil
		}
		return tx.Bucket(bucketCases).ForEach(func(k, v []byte) error {
			var c domain.AccessionCase
			if err := json.Unmarshal(v, &c); err != nil {
				return err
			}
			result = append(result, c)
			return nil
		})
	})
	sort.Slice(result, func(i, j int) bool {
		if result[i].UpdatedAt.Equal(result[j].UpdatedAt) {
			return result[i].ID < result[j].ID
		}
		return result[i].UpdatedAt.After(result[j].UpdatedAt)
	})
	return result, err
}

func loadTimeline(tx *bolt.Tx, caseID string) ([]audit.Event, error) {
	result := []audit.Event{}
	prefix := append([]byte(caseID), 0)
	cur := tx.Bucket(bucketAudit).Cursor()
	for k, v := cur.Seek(prefix); k != nil && bytes.HasPrefix(k, prefix); k, v = cur.Next() {
		var event audit.Event
		if err := json.Unmarshal(v, &event); err != nil {
			return nil, &domain.BusinessError{Code: domain.CodeCorrupt, Message: "审计事件记录损坏"}
		}
		result = append(result, event)
	}
	return audit.Stable(result), nil
}

func (s *Store) Timeline(caseID string) ([]audit.Event, error) {
	var events []audit.Event
	err := s.db.View(func(tx *bolt.Tx) error { var err error; events, err = loadTimeline(tx, caseID); return err })
	return events, err
}

func (s *Store) CertificateBySerial(serial uint64) (*domain.ReleaseCertificate, error) {
	var cert domain.ReleaseCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucketCertificates).Get(certificateKey(serial))
		if b == nil {
			return ErrNotFound
		}
		if err := json.Unmarshal(b, &cert); err != nil {
			return &domain.BusinessError{Code: domain.CodeCorrupt, Message: "放行凭据记录损坏"}
		}
		return nil
	})
	return &cert, err
}

func (s *Store) CertificateByDigest(digest string) (*domain.ReleaseCertificate, error) {
	var found *domain.ReleaseCertificate
	err := s.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucketCertificates).ForEach(func(_, value []byte) error {
			var cert domain.ReleaseCertificate
			if err := json.Unmarshal(value, &cert); err != nil {
				return &domain.BusinessError{Code: domain.CodeCorrupt, Message: "放行凭据记录损坏"}
			}
			if cert.ManifestDigest != digest {
				return nil
			}
			if found != nil {
				return domain.Conflict("清单摘要匹配多条放行凭据")
			}
			copy := cert
			found = &copy
			return nil
		})
	})
	if err != nil {
		return nil, err
	}
	if found == nil {
		return nil, ErrNotFound
	}
	return found, nil
}

func (s *Store) ManifestBySerial(serial uint64) (*domain.FrozenManifest, error) {
	var manifest domain.FrozenManifest
	err := s.db.View(func(tx *bolt.Tx) error {
		value := tx.Bucket(bucketManifests).Get([]byte(strconv.FormatUint(serial, 10)))
		if value == nil {
			return ErrNotFound
		}
		if err := json.Unmarshal(value, &manifest); err != nil {
			return &domain.BusinessError{Code: domain.CodeCorrupt, Message: "冻结清单记录损坏"}
		}
		return nil
	})
	return &manifest, err
}
