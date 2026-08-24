package storage

import (
	"context"
	"time"
)

func (s *Store) ListMedia(ctx context.Context, caseID string) ([]MediaObject, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT digest,case_id,content_type,size,stored_at FROM media_objects WHERE case_id=? ORDER BY stored_at,digest`, caseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []MediaObject{}
	for rows.Next() {
		var item MediaObject
		var stored string
		if err = rows.Scan(&item.Digest, &item.CaseID, &item.ContentType, &item.Size, &stored); err != nil {
			return nil, err
		}
		item.StoredAt, _ = time.Parse(time.RFC3339Nano, stored)
		items = append(items, item)
	}
	return items, rows.Err()
}

type IntegrityStatus struct {
	MediaReferences    int `json:"mediaReferences"`
	SegmentReferences  int `json:"segmentReferences"`
	ManifestReferences int `json:"manifestReferences"`
	MissingReferences  int `json:"missingReferences"`
}

func (s *Store) IntegrityStatus(ctx context.Context, caseID string) (IntegrityStatus, error) {
	var status IntegrityStatus
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM media_objects WHERE case_id=?`, caseID).Scan(&status.MediaReferences); err != nil {
		return status, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM segments WHERE case_id=?`, caseID).Scan(&status.SegmentReferences); err != nil {
		return status, err
	}
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM release_packages WHERE case_id=?`, caseID).Scan(&status.ManifestReferences); err != nil {
		return status, err
	}
	media, err := s.ListMedia(ctx, caseID)
	if err != nil {
		return status, err
	}
	for _, item := range media {
		if ok, checkErr := regularFile(s.objects + "/" + digestFilename(item.Digest)); checkErr != nil || !ok {
			status.MissingReferences++
		}
	}
	return status, nil
}
