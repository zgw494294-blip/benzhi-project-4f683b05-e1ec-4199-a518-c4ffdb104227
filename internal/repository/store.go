package repository

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"
)

type Store struct{ db *bolt.DB }

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o750); err != nil {
		return nil, err
	}
	db, err := bolt.Open(path, 0o600, &bolt.Options{Timeout: 2 * time.Second})
	if err != nil {
		return nil, fmt.Errorf("打开数据库: %w", err)
	}
	if err := db.Update(initialize); err != nil {
		db.Close()
		return nil, fmt.Errorf("初始化数据库: %w", err)
	}
	return &Store{db: db}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) View(fn func(*bolt.Tx) error) error   { return s.db.View(fn) }
func (s *Store) Update(fn func(*bolt.Tx) error) error { return s.db.Update(fn) }

func cloneBytes(in []byte) []byte {
	if in == nil {
		return nil
	}
	out := make([]byte, len(in))
	copy(out, in)
	return out
}

var ErrNotFound = errors.New("记录不存在")
var ErrRevision = errors.New("expectedRevision不匹配")
var ErrDuplicate = errors.New("记录已存在")
var ErrImmutable = errors.New("不可变记录禁止覆盖")
