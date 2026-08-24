package storage

import (
	"context"
	"database/sql"
)

type Tx struct {
	tx    *sql.Tx
	store *Store
}

func (s *Store) WithTx(ctx context.Context, fn func(*Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	wrapper := &Tx{tx: tx, store: s}
	if err = fn(wrapper); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
