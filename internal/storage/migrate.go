package storage

import (
	"context"
	"database/sql"
	"strings"
)

var errNoRows = sql.ErrNoRows

// enableForeignKeys re-enables foreign key enforcement on the connection that
// performed a table rebuild. It must run outside a transaction.
func enableForeignKeys(ctx context.Context, db *sql.DB) error {
	_, err := db.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
	return err
}

func (s *Store) migrate(ctx context.Context) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS schema_migrations(version INTEGER PRIMARY KEY, applied_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS cases(case_id TEXT PRIMARY KEY,title TEXT NOT NULL,alias TEXT NOT NULL,collector TEXT NOT NULL,status TEXT NOT NULL,version INTEGER NOT NULL,created_at TEXT NOT NULL,updated_at TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS consents(consent_id TEXT PRIMARY KEY,case_id TEXT NOT NULL UNIQUE,version INTEGER NOT NULL,audiences TEXT NOT NULL,purposes TEXT NOT NULL,embargo_until TEXT,withdrawal_terms TEXT NOT NULL,confirmed_by TEXT NOT NULL,confirmed_at TEXT NOT NULL,digest TEXT NOT NULL UNIQUE,FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS media_objects(digest TEXT NOT NULL,case_id TEXT NOT NULL,content_type TEXT NOT NULL,size INTEGER NOT NULL,stored_at TEXT NOT NULL,PRIMARY KEY(digest,case_id),FOREIGN KEY(case_id) REFERENCES cases(case_id))`,
		`CREATE TABLE IF NOT EXISTS segments(segment_id TEXT PRIMARY KEY,case_id TEXT NOT NULL,media_digest TEXT NOT NULL,speaker TEXT NOT NULL,start_ms INTEGER NOT NULL,end_ms INTEGER NOT NULL,transcript TEXT NOT NULL,tags TEXT NOT NULL,redaction_text TEXT NOT NULL DEFAULT '',decision_status TEXT NOT NULL,decision_reason TEXT NOT NULL DEFAULT '',reviewed_by TEXT NOT NULL DEFAULT '',FOREIGN KEY(case_id) REFERENCES cases(case_id),FOREIGN KEY(media_digest,case_id) REFERENCES media_objects(digest,case_id))`,
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
	// Bring any pre-existing database up to the composite-key schema so that
	// identical audio bytes can be referenced by multiple cases independently.
	if err := s.migrateMediaObjectsPrimaryKey(ctx); err != nil {
		return err
	}
	if err := s.migrateSegmentsForeignKey(ctx); err != nil {
		return err
	}
	return nil
}

// tableDefinition returns the CREATE TABLE SQL recorded by SQLite for the
// given table, or an empty string when the table does not exist.
func (s *Store) tableDefinition(ctx context.Context, table string) (string, error) {
	var sqlText string
	err := s.db.QueryRowContext(ctx, `SELECT name, sql FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(new(string), &sqlText)
	if err == errNoRows {
		return "", nil
	}
	return sqlText, err
}

// compactSQL removes whitespace so DDL fragments can be matched regardless of
// the spacing SQLite happened to preserve when storing the table definition.
func compactSQL(sqlText string) string {
	return strings.Join(strings.Fields(sqlText), "")
}

// rebuildTable recreates an existing table with a new definition while keeping
// its data. Foreign keys are disabled for the rebuild because the old table
// and its dependents may temporarily violate the new constraints.
func (s *Store) rebuildTable(ctx context.Context, table, definition, copyColumns string) error {
	if _, err := s.db.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return err
	}
	committed := false
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		_ = enableForeignKeys(ctx, s.db)
		return err
	}
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
		_ = enableForeignKeys(ctx, s.db)
	}()
	tmp := table + "_migration_new"
	if _, err = tx.ExecContext(ctx, `DROP TABLE IF EXISTS `+tmp); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, definition); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO `+tmp+`(`+copyColumns+`) SELECT `+copyColumns+` FROM `+table); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `DROP TABLE `+table); err != nil {
		return err
	}
	if _, err = tx.ExecContext(ctx, `ALTER TABLE `+tmp+` RENAME TO `+table); err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// migrateMediaObjectsPrimaryKey upgrades legacy databases whose media_objects
// table used digest as the sole primary key, which caused the second case
// uploading identical bytes to replace the first case's media reference.
func (s *Store) migrateMediaObjectsPrimaryKey(ctx context.Context) error {
	sqlText, err := s.tableDefinition(ctx, "media_objects")
	if err != nil {
		return err
	}
	if sqlText == "" {
		return nil
	}
	if strings.Contains(compactSQL(sqlText), "PRIMARYKEY(digest,case_id)") {
		return nil
	}
	definition := `CREATE TABLE media_objects_migration_new(digest TEXT NOT NULL,case_id TEXT NOT NULL,content_type TEXT NOT NULL,size INTEGER NOT NULL,stored_at TEXT NOT NULL,PRIMARY KEY(digest,case_id),FOREIGN KEY(case_id) REFERENCES cases(case_id))`
	if err = s.rebuildTable(ctx, "media_objects", definition, "digest,case_id,content_type,size,stored_at"); err != nil {
		return err
	}
	return nil
}

// migrateSegmentsForeignKey upgrades legacy databases whose segments table
// referenced media_objects(digest) alone, which can no longer resolve once
// media_objects uses a composite primary key.
func (s *Store) migrateSegmentsForeignKey(ctx context.Context) error {
	sqlText, err := s.tableDefinition(ctx, "segments")
	if err != nil {
		return err
	}
	if sqlText == "" {
		return nil
	}
	if strings.Contains(compactSQL(sqlText), "FOREIGNKEY(media_digest,case_id)REFERENCESmedia_objects(digest,case_id)") {
		return nil
	}
	definition := `CREATE TABLE segments_migration_new(segment_id TEXT PRIMARY KEY,case_id TEXT NOT NULL,media_digest TEXT NOT NULL,speaker TEXT NOT NULL,start_ms INTEGER NOT NULL,end_ms INTEGER NOT NULL,transcript TEXT NOT NULL,tags TEXT NOT NULL,redaction_text TEXT NOT NULL DEFAULT '',decision_status TEXT NOT NULL,decision_reason TEXT NOT NULL DEFAULT '',reviewed_by TEXT NOT NULL DEFAULT '',FOREIGN KEY(case_id) REFERENCES cases(case_id),FOREIGN KEY(media_digest,case_id) REFERENCES media_objects(digest,case_id))`
	if err = s.rebuildTable(ctx, "segments", definition, "segment_id,case_id,media_digest,speaker,start_ms,end_ms,transcript,tags,redaction_text,decision_status,decision_reason,reviewed_by"); err != nil {
		return err
	}
	// Recreate the index that was dropped with the legacy table.
	if _, err = s.db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_segments_case ON segments(case_id,start_ms)`); err != nil {
		return err
	}
	return nil
}
