package storage

import (
	"context"
	"database/sql"
	"oralarchive/internal/domain"
	"time"
)

type scanner interface{ Scan(...any) error }

func scanCase(row scanner) (domain.OralHistoryCase, error) {
	var c domain.OralHistoryCase
	var status string
	var created, updated string
	err := row.Scan(&c.CaseID, &c.Title, &c.IntervieweeAlias, &c.Collector, &status, &c.Version, &created, &updated)
	if err != nil {
		return c, err
	}
	c.Status = domain.CaseStatus(status)
	c.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	c.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
	return c, nil
}
func (s *Store) CreateCase(ctx context.Context, c domain.OralHistoryCase, actor string) error {
	return s.WithTx(ctx, func(tx *Tx) error {
		if err := tx.InsertCase(ctx, c); err != nil {
			return err
		}
		return tx.AppendAudit(ctx, domain.AuditEvent{CaseID: c.CaseID, Action: "CASE_CREATED", Actor: actor, Detail: "建立口述史采集案卷", Version: c.Version, OccurredAt: c.CreatedAt})
	})
}
func (tx *Tx) InsertCase(ctx context.Context, c domain.OralHistoryCase) error {
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO cases(case_id,title,alias,collector,status,version,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)`, c.CaseID, c.Title, c.IntervieweeAlias, c.Collector, c.Status, c.Version, c.CreatedAt.Format(time.RFC3339Nano), c.UpdatedAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) GetCase(ctx context.Context, id string) (domain.OralHistoryCase, error) {
	c, err := scanCase(s.db.QueryRowContext(ctx, `SELECT case_id,title,alias,collector,status,version,created_at,updated_at FROM cases WHERE case_id=?`, id))
	if err == sql.ErrNoRows {
		return c, ErrNotFound
	}
	return c, err
}
func (tx *Tx) GetCase(ctx context.Context, id string) (domain.OralHistoryCase, error) {
	c, err := scanCase(tx.tx.QueryRowContext(ctx, `SELECT case_id,title,alias,collector,status,version,created_at,updated_at FROM cases WHERE case_id=?`, id))
	if err == sql.ErrNoRows {
		return c, ErrNotFound
	}
	return c, err
}
func (tx *Tx) UpdateCase(ctx context.Context, c domain.OralHistoryCase, expected int64) error {
	res, err := tx.tx.ExecContext(ctx, `UPDATE cases SET title=?,alias=?,collector=?,status=?,version=?,updated_at=? WHERE case_id=? AND version=?`, c.Title, c.IntervieweeAlias, c.Collector, c.Status, c.Version, c.UpdatedAt.Format(time.RFC3339Nano), c.CaseID, expected)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrConflict
	}
	return nil
}
func (s *Store) ListCases(ctx context.Context) ([]domain.OralHistoryCase, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT case_id,title,alias,collector,status,version,created_at,updated_at FROM cases ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.OralHistoryCase{}
	for rows.Next() {
		c, err := scanCase(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
