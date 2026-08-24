package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"oralarchive/internal/domain"
	"time"
)

func (tx *Tx) InsertConfirmation(ctx context.Context, c domain.Confirmation) error {
	ids, _ := json.Marshal(c.ReturnedSegmentIDs)
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO confirmations(case_id,confirmed,returned_segments,comment,actor,decided_at) VALUES(?,?,?,?,?,?)`, c.CaseID, c.Confirmed, ids, c.Comment, c.Actor, c.DecidedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) LatestConfirmation(ctx context.Context, caseID string) (*domain.Confirmation, error) {
	var c domain.Confirmation
	var yes int
	var ids []byte
	var at string
	err := s.db.QueryRowContext(ctx, `SELECT case_id,confirmed,returned_segments,comment,actor,decided_at FROM confirmations WHERE case_id=? ORDER BY id DESC LIMIT 1`, caseID).Scan(&c.CaseID, &yes, &ids, &c.Comment, &c.Actor, &at)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	c.Confirmed = yes == 1
	_ = json.Unmarshal(ids, &c.ReturnedSegmentIDs)
	c.DecidedAt, _ = time.Parse(time.RFC3339Nano, at)
	return &c, nil
}

func (s *Store) ListConfirmations(ctx context.Context, caseID string) ([]domain.Confirmation, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT case_id,confirmed,returned_segments,comment,actor,decided_at FROM confirmations WHERE case_id=? ORDER BY id DESC`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.Confirmation
	for rows.Next() {
		var c domain.Confirmation
		var yes int
		var ids []byte
		var at string
		if err = rows.Scan(&c.CaseID, &yes, &ids, &c.Comment, &c.Actor, &at); err != nil {
			return nil, err
		}
		c.Confirmed = yes == 1
		if err = json.Unmarshal(ids, &c.ReturnedSegmentIDs); err != nil {
			return nil, err
		}
		c.DecidedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
