package storage

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"oralarchive/internal/domain"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type MediaObject struct {
	Digest      string    `json:"digest"`
	CaseID      string    `json:"caseID"`
	ContentType string    `json:"contentType"`
	Size        int64     `json:"size"`
	StoredAt    time.Time `json:"storedAt"`
}

func digestFilename(d string) string { return strings.TrimPrefix(d, "sha256:") }
func (s *Store) PutMedia(ctx context.Context, caseID, contentType string, r io.Reader, max int64) (MediaObject, error) {
	limited := io.LimitReader(r, max+1)
	b, err := io.ReadAll(limited)
	if err != nil {
		return MediaObject{}, err
	}
	if int64(len(b)) > max {
		return MediaObject{}, fmt.Errorf("上传超过 %d 字节限制", max)
	}
	digest := domain.BytesDigest(b)
	path := filepath.Join(s.objects, digestFilename(digest))
	if err = atomicWrite(path, b, 0600); err != nil {
		return MediaObject{}, err
	}
	obj := MediaObject{digest, caseID, contentType, int64(len(b)), time.Now().UTC()}
	// Identical audio bytes are shared across cases: the underlying object file
	// is written once, while each case keeps its own media_objects reference via
	// the composite primary key (digest, case_id). INSERT OR IGNORE preserves an
	// existing reference for this case instead of replacing another case's row.
	if _, err = s.db.ExecContext(ctx, `INSERT OR IGNORE INTO media_objects(digest,case_id,content_type,size,stored_at) VALUES(?,?,?,?,?)`, obj.Digest, obj.CaseID, obj.ContentType, obj.Size, obj.StoredAt.Format(time.RFC3339Nano)); err != nil {
		return obj, err
	}
	// Re-read the canonical row so callers always see the stored reference for
	// this case (which may predate this upload) rather than a transient copy.
	var storedAt string
	if qerr := s.db.QueryRowContext(ctx, `SELECT content_type,size,stored_at FROM media_objects WHERE digest=? AND case_id=?`, obj.Digest, obj.CaseID).Scan(&obj.ContentType, &obj.Size, &storedAt); qerr == nil {
		obj.StoredAt, _ = time.Parse(time.RFC3339Nano, storedAt)
	}
	return obj, nil
}
func (s *Store) HasMedia(ctx context.Context, digest, caseID string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, `SELECT 1 FROM media_objects WHERE digest=? AND case_id=?`, digest, caseID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (tx *Tx) HasMedia(ctx context.Context, digest, caseID string) (bool, error) {
	var one int
	err := tx.tx.QueryRowContext(ctx, `SELECT 1 FROM media_objects WHERE digest=? AND case_id=?`, digest, caseID).Scan(&one)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func (s *Store) MediaFileExists(ctx context.Context, digest string) (bool, error) {
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}
	ok, err := regularFile(filepath.Join(s.objects, digestFilename(digest)))
	if os.IsNotExist(err) {
		return false, nil
	}
	return ok, err
}
func atomicWrite(path string, b []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".pending-")
	if err != nil {
		return err
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(b); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Chmod(mode); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if _, err = os.Stat(path); err == nil {
		return nil
	}
	return os.Rename(name, path)
}
