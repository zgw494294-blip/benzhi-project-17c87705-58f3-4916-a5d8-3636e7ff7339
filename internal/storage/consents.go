package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"oralarchive/internal/domain"
	"time"
)

func (tx *Tx) InsertConsent(ctx context.Context, c domain.ConsentScope) error {
	a, _ := json.Marshal(c.AllowedAudiences)
	p, _ := json.Marshal(c.AllowedPurposes)
	var embargo any
	if c.EmbargoUntil != nil {
		embargo = c.EmbargoUntil.Format(time.RFC3339Nano)
	}
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO consents(consent_id,case_id,version,audiences,purposes,embargo_until,withdrawal_terms,confirmed_by,confirmed_at,digest) VALUES(?,?,?,?,?,?,?,?,?,?)`, c.ConsentID, c.CaseID, c.Version, a, p, embargo, c.WithdrawalTerms, c.ConfirmedBy, c.ConfirmedAt.Format(time.RFC3339Nano), c.Digest)
	return err
}
func scanConsent(row scanner) (domain.ConsentScope, error) {
	var c domain.ConsentScope
	var a, p []byte
	var embargo sql.NullString
	var confirmed string
	err := row.Scan(&c.ConsentID, &c.CaseID, &c.Version, &a, &p, &embargo, &c.WithdrawalTerms, &c.ConfirmedBy, &confirmed, &c.Digest)
	if err != nil {
		return c, err
	}
	_ = json.Unmarshal(a, &c.AllowedAudiences)
	_ = json.Unmarshal(p, &c.AllowedPurposes)
	c.ConfirmedAt, _ = time.Parse(time.RFC3339Nano, confirmed)
	if embargo.Valid {
		t, _ := time.Parse(time.RFC3339Nano, embargo.String)
		c.EmbargoUntil = &t
	}
	return c, nil
}
func (s *Store) GetConsent(ctx context.Context, caseID string) (*domain.ConsentScope, error) {
	c, err := scanConsent(s.db.QueryRowContext(ctx, `SELECT consent_id,case_id,version,audiences,purposes,embargo_until,withdrawal_terms,confirmed_by,confirmed_at,digest FROM consents WHERE case_id=?`, caseID))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &c, err
}
func (tx *Tx) GetConsent(ctx context.Context, caseID string) (*domain.ConsentScope, error) {
	c, err := scanConsent(tx.tx.QueryRowContext(ctx, `SELECT consent_id,case_id,version,audiences,purposes,embargo_until,withdrawal_terms,confirmed_by,confirmed_at,digest FROM consents WHERE case_id=?`, caseID))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &c, err
}
