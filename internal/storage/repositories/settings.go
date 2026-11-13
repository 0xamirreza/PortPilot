package repositories

import (
	"context"
	"database/sql"
	"time"
)

// Settings mirrors selected preferences into SQLite.
// The JSON config file remains the file the operator edits.
type Settings struct {
	db *sql.DB
}

// NewSettings binds the repository to an open database.
func NewSettings(db *sql.DB) *Settings {
	return &Settings{db: db}
}

// Put stores one setting.
func (s *Settings) Put(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO settings (key, value, updated_at) VALUES (?, ?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value, updated_at = excluded.updated_at`,
		key, value, time.Now().UTC().Format(time.RFC3339Nano))
	return err
}
