// Package sqlite opens the embedded database and checks its integrity.
package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/0xamirreza/portpilot/internal/storage/migrations"

	_ "modernc.org/sqlite"
)

// Open creates or opens the database, checks integrity, and migrates it.
// A corrupt file is renamed aside and replaced. The caller owns db.
func Open(ctx context.Context, path string) (*sql.DB, error) {
	if err := backup(path); err != nil {
		return nil, err
	}
	db, err := connect(path)
	if err != nil {
		return nil, err
	}
	ok, err := integrity(ctx, db)
	if err != nil {
		_ = db.Close()
		return nil, err
	}
	if !ok {
		_ = db.Close()
		if err := quarantine(path); err != nil {
			return nil, err
		}
		db, err = connect(path)
		if err != nil {
			return nil, err
		}
	}
	if err := migrations.Apply(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := chmod(path); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func connect(path string) (*sql.DB, error) {
	q := url.Values{}
	q.Add("_pragma", "busy_timeout(5000)")
	q.Add("_pragma", "foreign_keys(1)")
	q.Add("_pragma", "journal_mode(WAL)")
	dsn := (&url.URL{Scheme: "file", Path: path, RawQuery: q.Encode()}).String()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	return db, nil
}

func integrity(ctx context.Context, db *sql.DB) (bool, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA integrity_check`)
	if err != nil {
		return false, fmt.Errorf("integrity_check: %w", err)
	}
	defer rows.Close()
	var messages []string
	for rows.Next() {
		var msg string
		if err := rows.Scan(&msg); err != nil {
			return false, err
		}
		messages = append(messages, msg)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	return len(messages) == 1 && messages[0] == "ok", nil
}

func backup(path string) error {
	src, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer src.Close()
	dst, err := os.OpenFile(path+".bak", os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("backup database: %w", err)
	}
	_, copyErr := io.Copy(dst, src)
	closeErr := dst.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}

func quarantine(path string) error {
	stamp := time.Now().UTC().Format("20060102T150405Z")
	for _, suffix := range []string{"", "-wal", "-shm"} {
		from := path + suffix
		if _, err := os.Stat(from); err != nil {
			continue
		}
		to := fmt.Sprintf("%s.corrupt-%s%s", path, stamp, suffix)
		if err := os.Rename(from, to); err != nil {
			return fmt.Errorf("quarantine %s: %w", from, err)
		}
	}
	return nil
}

func chmod(path string) error {
	for _, suffix := range []string{"", "-wal", "-shm"} {
		p := path + suffix
		if _, err := os.Stat(p); err != nil {
			continue
		}
		if err := os.Chmod(p, 0o600); err != nil {
			return err
		}
	}
	if err := os.Chmod(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return nil
}
