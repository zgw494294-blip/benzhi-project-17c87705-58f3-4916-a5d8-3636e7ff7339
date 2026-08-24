package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"oralarchive/internal/domain"
	"os"
	"path/filepath"
	"time"
)

func (tx *Tx) InsertPackage(ctx context.Context, p domain.ReleasePackage, manifest []byte) error {
	if err := atomicWrite(filepath.Join(tx.store.manifests, digestFilename(p.ManifestDigest)+".json"), manifest, 0600); err != nil {
		return err
	}
	digests, _ := json.Marshal(p.SegmentDigests)
	_, err := tx.tx.ExecContext(ctx, `INSERT INTO release_packages(package_id,case_id,manifest_digest,consent_digest,segment_digests,issued_at,issued_by) VALUES(?,?,?,?,?,?,?)`, p.PackageID, p.CaseID, p.ManifestDigest, p.ConsentDigest, digests, p.IssuedAt.Format(time.RFC3339Nano), p.IssuedBy)
	return err
}
func scanPackage(row scanner) (domain.ReleasePackage, error) {
	var p domain.ReleasePackage
	var ds []byte
	var at string
	err := row.Scan(&p.PackageID, &p.CaseID, &p.ManifestDigest, &p.ConsentDigest, &ds, &at, &p.IssuedBy)
	_ = json.Unmarshal(ds, &p.SegmentDigests)
	p.IssuedAt, _ = time.Parse(time.RFC3339Nano, at)
	return p, err
}
func (s *Store) GetPackageByCase(ctx context.Context, caseID string) (*domain.ReleasePackage, error) {
	p, err := scanPackage(s.db.QueryRowContext(ctx, `SELECT package_id,case_id,manifest_digest,consent_digest,segment_digests,issued_at,issued_by FROM release_packages WHERE case_id=?`, caseID))
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	return &p, err
}
func (s *Store) VerifyReferences(ctx context.Context) error {
	rows, err := s.db.QueryContext(ctx, `SELECT digest FROM media_objects`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		if err = rows.Scan(&d); err != nil {
			return err
		}
		if _, err = os.Stat(filepath.Join(s.objects, digestFilename(d))); err != nil {
			return err
		}
	}
	rows.Close()
	rows, err = s.db.QueryContext(ctx, `SELECT manifest_digest FROM release_packages`)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var d string
		if err = rows.Scan(&d); err != nil {
			return err
		}
		if _, err = os.Stat(filepath.Join(s.manifests, digestFilename(d)+".json")); err != nil {
			return err
		}
	}
	return rows.Err()
}
