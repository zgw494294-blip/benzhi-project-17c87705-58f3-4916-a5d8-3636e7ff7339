package storage

import (
	"context"
	"database/sql"
	"time"
)

func (s *Store) GetIdempotency(ctx context.Context, key, operation, caseID string) ([]byte, bool, error) {
	var response []byte
	err := s.db.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE key=? AND operation=? AND case_id=?`, key, operation, caseID).Scan(&response)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return response, err == nil, err
}
func (tx *Tx) GetIdempotency(ctx context.Context, key, operation, caseID string) ([]byte, bool, error) {
	var response []byte
	err := tx.tx.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE key=? AND operation=? AND case_id=?`, key, operation, caseID).Scan(&response)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return response, err == nil, err
}
func (tx *Tx) PutIdempotency(ctx context.Context, key, operation, caseID string, response []byte) error {
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO idempotency_keys(key,operation,case_id,response,created_at) VALUES(?,?,?,?,?)`, key, operation, caseID, response, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
func (s *Store) GetIdempotencyByKey(ctx context.Context, key, operation string) ([]byte, bool, error) {
	var response []byte
	err := s.db.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE key=? AND operation=?`, key, operation).Scan(&response)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return response, err == nil, err
}
func (tx *Tx) GetIdempotencyByKey(ctx context.Context, key, operation string) ([]byte, bool, error) {
	var response []byte
	err := tx.tx.QueryRowContext(ctx, `SELECT response FROM idempotency_keys WHERE key=? AND operation=?`, key, operation).Scan(&response)
	if err == sql.ErrNoRows {
		return nil, false, nil
	}
	return response, err == nil, err
}
