package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"oralarchive/internal/domain"
)

func (s *Store) InsertSegment(ctx context.Context, seg domain.TranscriptSegment) error {
	tags, _ := json.Marshal(seg.SensitivityTags)
	_, err := s.db.ExecContext(ctx, `INSERT INTO segments(segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, seg.SegmentID, seg.CaseID, seg.MediaDigest, seg.Speaker, seg.StartMillis, seg.EndMillis, seg.Transcript, tags, seg.RedactionText, seg.DecisionStatus, seg.DecisionReason, seg.ReviewedBy)
	return err
}
func (tx *Tx) InsertSegment(ctx context.Context, seg domain.TranscriptSegment) error {
	tags, _ := json.Marshal(seg.SensitivityTags)
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO segments(segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, seg.SegmentID, seg.CaseID, seg.MediaDigest, seg.Speaker, seg.StartMillis, seg.EndMillis, seg.Transcript, tags, seg.RedactionText, seg.DecisionStatus, seg.DecisionReason, seg.ReviewedBy)
	return err
}

func (tx *Tx) GetSegment(ctx context.Context, id string) (domain.TranscriptSegment, error) {
	seg, err := scanSegment(tx.tx.QueryRowContext(ctx, `SELECT segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by FROM segments WHERE segment_id=?`, id))
	if err == sql.ErrNoRows {
		return seg, ErrNotFound
	}
	return seg, err
}

func (tx *Tx) ListSegments(ctx context.Context, caseID string) ([]domain.TranscriptSegment, error) {
	rows, err := tx.tx.QueryContext(ctx, `SELECT segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by FROM segments WHERE case_id=? ORDER BY start_ms,segment_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TranscriptSegment{}
	for rows.Next() {
		seg, scanErr := scanSegment(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}
func scanSegment(row scanner) (domain.TranscriptSegment, error) {
	var s domain.TranscriptSegment
	var tags []byte
	var decision string
	err := row.Scan(&s.SegmentID, &s.CaseID, &s.MediaDigest, &s.Speaker, &s.StartMillis, &s.EndMillis, &s.Transcript, &tags, &s.RedactionText, &decision, &s.DecisionReason, &s.ReviewedBy)
	_ = json.Unmarshal(tags, &s.SensitivityTags)
	s.DecisionStatus = domain.DecisionStatus(decision)
	return s, err
}
func (s *Store) GetSegment(ctx context.Context, id string) (domain.TranscriptSegment, error) {
	seg, err := scanSegment(s.db.QueryRowContext(ctx, `SELECT segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by FROM segments WHERE segment_id=?`, id))
	if err == sql.ErrNoRows {
		return seg, ErrNotFound
	}
	return seg, err
}
func (s *Store) ListSegments(ctx context.Context, caseID string) ([]domain.TranscriptSegment, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by FROM segments WHERE case_id=? ORDER BY start_ms,segment_id`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.TranscriptSegment{}
	for rows.Next() {
		s, err := scanSegment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
func (s *Store) UpdateSegmentDecision(ctx context.Context, seg domain.TranscriptSegment) error {
	res, err := s.db.ExecContext(ctx, `UPDATE segments SET redaction_text=?,decision_status=?,decision_reason=?,reviewed_by=? WHERE segment_id=? AND case_id=?`, seg.RedactionText, seg.DecisionStatus, seg.DecisionReason, seg.ReviewedBy, seg.SegmentID, seg.CaseID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrNotFound
	}
	return nil
}
func (tx *Tx) UpdateSegmentDecision(ctx context.Context, seg domain.TranscriptSegment) error {
	res, err := tx.tx.ExecContext(ctx, `UPDATE segments SET redaction_text=?,decision_status=?,decision_reason=?,reviewed_by=? WHERE segment_id=? AND case_id=?`, seg.RedactionText, seg.DecisionStatus, seg.DecisionReason, seg.ReviewedBy, seg.SegmentID, seg.CaseID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrNotFound
	}
	return nil
}

func (tx *Tx) UpdateSegmentRemediation(ctx context.Context, seg domain.TranscriptSegment) error {
	res, err := tx.tx.ExecContext(ctx, `UPDATE segments SET redaction_text=?,decision_status=?,decision_reason=?,reviewed_by=? WHERE segment_id=? AND case_id=?`, seg.RedactionText, seg.DecisionStatus, seg.DecisionReason, seg.ReviewedBy, seg.SegmentID, seg.CaseID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrNotFound
	}
	return nil
}

func (tx *Tx) UpdateSegmentTiming(ctx context.Context, seg domain.TranscriptSegment) error {
	res, err := tx.tx.ExecContext(ctx, `UPDATE segments SET media_digest=?,start_ms=?,end_ms=? WHERE segment_id=? AND case_id=?`, seg.MediaDigest, seg.StartMillis, seg.EndMillis, seg.SegmentID, seg.CaseID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n != 1 {
		return ErrNotFound
	}
	return nil
}
