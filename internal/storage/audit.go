package storage

import (
	"context"
	"oralarchive/internal/domain"
	"time"
)

func (tx *Tx) AppendAudit(ctx context.Context, e domain.AuditEvent) error {
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO audit_events(case_id,action,actor,detail,version,occurred_at) VALUES(?,?,?,?,?,?)`, e.CaseID, e.Action, e.Actor, e.Detail, e.Version, e.OccurredAt.Format(time.RFC3339Nano))
	return err
}
func (s *Store) AppendAudit(ctx context.Context, e domain.AuditEvent) error {
	return s.WithTx(ctx, func(tx *Tx) error { return tx.AppendAudit(ctx, e) })
}
func (s *Store) ListAudit(ctx context.Context, caseID string) ([]domain.AuditEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id,case_id,action,actor,detail,version,occurred_at FROM audit_events WHERE case_id=? ORDER BY event_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.AuditEvent{}
	for rows.Next() {
		var e domain.AuditEvent
		var at string
		if err = rows.Scan(&e.EventID, &e.CaseID, &e.Action, &e.Actor, &e.Detail, &e.Version, &at); err != nil {
			return nil, err
		}
		e.OccurredAt, _ = time.Parse(time.RFC3339Nano, at)
		out = append(out, e)
	}
	return out, rows.Err()
}
