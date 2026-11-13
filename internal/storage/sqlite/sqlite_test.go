package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
)

func TestOpenMigratesAndForeignKeys(t *testing.T) {
	path := filepath.Join(t.TempDir(), "portpilot.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(`SELECT MAX(version) FROM schema_migrations`).Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 2 {
		t.Fatalf("version = %d", version)
	}
	var fk int
	if err := db.QueryRow(`PRAGMA foreign_keys`).Scan(&fk); err != nil {
		t.Fatal(err)
	}
	if fk != 1 {
		t.Fatalf("foreign_keys = %d", fk)
	}
	if _, err := db.Exec(`INSERT INTO scheduled_rules (rule_id, expires_at, mechanism, status) VALUES (99, 'x', 'none', 'pending')`); err == nil {
		t.Fatal("foreign key was not enforced")
	}
}

func TestReopenKeepsSchema(t *testing.T) {
	path := filepath.Join(t.TempDir(), "portpilot.db")
	db, err := Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO settings (key, value, updated_at) VALUES ('theme', 'dark', 't')`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	db, err = Open(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var value string
	if err := db.QueryRow(`SELECT value FROM settings WHERE key = 'theme'`).Scan(&value); err != nil {
		t.Fatal(err)
	}
	if value != "dark" {
		t.Fatal(value)
	}
	_ = sql.ErrNoRows
}
