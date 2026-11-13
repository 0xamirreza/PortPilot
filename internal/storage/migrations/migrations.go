// Package migrations applies the embedded SQLite schema.
package migrations

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"time"
)

//go:embed 001_init.sql
var initSQL string

//go:embed 002_rule_state.sql
var ruleStateSQL string

const version = 2

type step struct {
	version int
	name    string
	sql     string
}

// Apply runs pending migrations inside a transaction.
func Apply(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		name TEXT NOT NULL,
		applied_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("migration table: %w", err)
	}
	var current int
	err := db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current)
	if err != nil {
		return fmt.Errorf("migration version: %w", err)
	}
	steps := []step{
		{1, "001_init", initSQL},
		{2, "002_rule_state", ruleStateSQL},
	}
	if current >= version {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, s := range steps {
		if s.version <= current {
			continue
		}
		if _, err := tx.ExecContext(ctx, s.sql); err != nil {
			return fmt.Errorf("apply schema v%d: %w", s.version, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, name, applied_at) VALUES (?, ?, ?)`,
			s.version, s.name, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
			return err
		}
	}
	return tx.Commit()
}
