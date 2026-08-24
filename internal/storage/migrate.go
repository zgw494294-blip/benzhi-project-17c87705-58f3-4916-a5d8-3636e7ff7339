package storage

import "context"

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS cases(case_id TEXT PRIMARY KEY,title TEXT NOT NULL,alias TEXT NOT NULL,collector TEXT NOT NULL,status TEXT NOT NULL,version INTEGER NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS consents(consent_id TEXT PRIMARY KEY,case_id TEXT NOT NULL UNIQUE,version INTEGER NOT NULL,audiences TEXT NOT NULL,purposes TEXT NOT NULL,embargo_until TEXT,withdrawal_terms TEXT NOT NULL,confirmed_by TEXT NOT NULL,confirmed_at TEXT NOT NULL,digest TEXT NOT NULL UNIQUE,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS media_objects(digest TEXT PRIMARY KEY,case_id TEXT NOT NULL,content_type TEXT NOT NULL,size INTEGER NOT NULL,stored_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS segments(segment_id TEXT PRIMARY KEY,case_id TEXT NOT NULL,media_digest TEXT NOT NULL,speaker TEXT NOT NULL,start_ms INTEGER NOT NULL,end_ms INTEGER NOT NULL,transcript TEXT NOT NULL,tags TEXT NOT NULL,redaction_text TEXT NOT NULL DEFAULT '',decision_status TEXT NOT NULL,decision_reason TEXT NOT NULL DEFAULT '',reviewed_by TEXT NOT NULL DEFAULT '',FOREIGN KEY(case_id) REFERENCES cases(case_id),FOREIGN KEY(media_digest) REFERENCES media_objects(digest))`,
		`CREATE TABLE IF NOT EXISTS confirmations(id INTEGER PRIMARY KEY AUTOINCREMENT,case_id TEXT NOT NULL,confirmed INTEGER NOT NULL,returned_segments TEXT NOT NULL,comment TEXT NOT NULL,actor TEXT NOT NULL,decided_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS release_packages(package_id TEXT PRIMARY KEY,case_id TEXT NOT NULL UNIQUE,manifest_digest TEXT NOT NULL UNIQUE,consent_digest TEXT NOT NULL,segment_digests TEXT NOT NULL,issued_at TEXT NOT NULL,issued_by TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS audit_events(event_id INTEGER PRIMARY KEY AUTOINCREMENT,case_id TEXT NOT NULL,action TEXT NOT NULL,actor TEXT NOT NULL,detail TEXT NOT NULL,version INTEGER NOT NULL,occurred_at TEXT NOT NULL,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS idempotency_keys(key TEXT PRIMARY KEY,operation TEXT NOT NULL,case_id TEXT NOT NULL,response TEXT NOT NULL,created_at TEXT NOT NULL)`,
		`CREATE INDEX IF NOT EXISTS idx_segments_case ON segments(case_id,start_ms)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_case ON audit_events(case_id,event_id)`,
		`INSERT OR IGNORE INTO schema_migrations(version,applied_at) VALUES(1,datetime('now'))`,
	}
	for _, q := range statements {
		if _, err := s.db.ExecContext(ctx, q); err != nil {
			return err
		}
	}
	return nil
}
