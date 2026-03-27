package db

import (
	"os"
	"path/filepath"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"
)

func Open(path string) (*sqlx.DB, error) {
	// Only create directory if path contains a directory component
	dir := filepath.Dir(path)
	if dir != "" && dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sqlx.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := Migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func Migrate(db *sqlx.DB) error {
	schema := `
PRAGMA foreign_keys = ON;

CREATE TABLE IF NOT EXISTS contacts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    gender TEXT NOT NULL DEFAULT 'unknown',
    tags TEXT NOT NULL DEFAULT '',
    profile_summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS sessions (
    id TEXT PRIMARY KEY,
    contact_id TEXT NOT NULL,
    parent_session_id TEXT,
    status TEXT NOT NULL,
    summary TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL,
    FOREIGN KEY(contact_id) REFERENCES contacts(id)
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL,
    speaker TEXT NOT NULL,
    content TEXT NOT NULL,
    emotion TEXT NOT NULL DEFAULT '',
    intent TEXT NOT NULL DEFAULT '',
    timestamp DATETIME NOT NULL,
    FOREIGN KEY(session_id) REFERENCES sessions(id)
);

CREATE TABLE IF NOT EXISTS raw_logs (
    id TEXT PRIMARY KEY,
    contact_id TEXT NOT NULL,
    raw_text TEXT NOT NULL,
    created_at DATETIME NOT NULL,
    FOREIGN KEY(contact_id) REFERENCES contacts(id)
);

CREATE INDEX IF NOT EXISTS idx_sessions_contact_status ON sessions(contact_id, status, updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_messages_session_time ON messages(session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_raw_logs_contact_time ON raw_logs(contact_id, created_at DESC);
`
	_, err := db.Exec(schema)
	return err
}
