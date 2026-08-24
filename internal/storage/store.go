package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func regularFile(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.Mode().IsRegular(), nil
}

var ErrNotFound = errors.New("记录不存在")
var ErrConflict = errors.New("版本冲突")

type Store struct {
	db        *sql.DB
	root      string
	objects   string
	manifests string
}

func Open(ctx context.Context, root string) (*Store, error) {
	if err := os.MkdirAll(root, 0700); err != nil {
		return nil, err
	}
	objects := filepath.Join(root, "objects")
	manifests := filepath.Join(root, "manifests")
	if err := os.MkdirAll(objects, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(manifests, 0700); err != nil {
		return nil, err
	}
	dsn := "file:" + filepath.Join(root, "oralarchive.db") + "?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=foreign_keys(ON)"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	s := &Store{db: db, root: root, objects: objects, manifests: manifests}
	if err = s.migrate(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("迁移数据库: %w", err)
	}
	if err = s.VerifyReferences(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("校验内容引用: %w", err)
	}
	return s, nil
}
func (s *Store) Close() error { return s.db.Close() }
func (s *Store) DB() *sql.DB  { return s.db }
