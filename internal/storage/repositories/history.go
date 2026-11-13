// Package repositories stores PortPilot's own records.
package repositories

import (
	"context"
	"database/sql"
	"time"
)

// Entry is one audit row. It records what PortPilot did, not kernel state.
type Entry struct {
	ID         int64
	RecordedAt time.Time
	Action     string
	Backend    string
	Protocol   string
	Port       int
	Target     string
	Duration   string
	Result     string
	Detail     string
	User       string
}

// History reads and writes action_history.
type History struct {
	db *sql.DB
}

// NewHistory binds the repository to an open database.
func NewHistory(db *sql.DB) *History {
	return &History{db: db}
}

// Insert appends one audit row.
func (h *History) Insert(ctx context.Context, e Entry) error {
	if e.RecordedAt.IsZero() {
		e.RecordedAt = time.Now()
	}
	_, err := h.db.ExecContext(ctx, `INSERT INTO action_history
		(recorded_at, action, backend, protocol, port, target, duration, result, detail, user_name)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.RecordedAt.UTC().Format(time.RFC3339Nano), e.Action, e.Backend, e.Protocol, e.Port, e.Target, e.Duration, e.Result, e.Detail, e.User)
	return err
}

// Recent returns the newest rows, limited to n.
func (h *History) Recent(ctx context.Context, n int) ([]Entry, error) {
	if n <= 0 {
		n = 200
	}
	rows, err := h.db.QueryContext(ctx, `SELECT id, recorded_at, action, backend, protocol, port, target, duration, result, detail, user_name
		FROM action_history ORDER BY id DESC LIMIT ?`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var at string
		if err := rows.Scan(&e.ID, &at, &e.Action, &e.Backend, &e.Protocol, &e.Port, &e.Target, &e.Duration, &e.Result, &e.Detail, &e.User); err != nil {
			return nil, err
		}
		e.RecordedAt, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
