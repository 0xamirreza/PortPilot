package repositories

import (
	"context"
	"database/sql"
)

// OwnedRule is a PortPilot rule remembered locally. The kernel copy is read separately.
type OwnedRule struct {
	Identity    string
	Backend     string
	Protocol    string
	Port        int
	Action      string
	Source      string
	Iface       string
	Family      string
	DurationSec int
	Status      string
}

// Rules stores owned-rule specs so a disabled rule can be enabled again.
type Rules struct {
	db *sql.DB
}

// NewRules binds the repository.
func NewRules(db *sql.DB) *Rules { return &Rules{db: db} }

// Upsert records an active or disabled spec.
func (r *Rules) Upsert(ctx context.Context, rule OwnedRule) error {
	if rule.Status == "" {
		rule.Status = "active"
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO firewall_rules
		(backend, identity, proto, port, action, comment, created_at, status, source, iface, family, duration_sec)
		VALUES (?, ?, ?, ?, ?, ?, datetime('now'), ?, ?, ?, ?, ?)
		ON CONFLICT(identity) DO UPDATE SET
			status=excluded.status, backend=excluded.backend, proto=excluded.proto, port=excluded.port,
			action=excluded.action, source=excluded.source, iface=excluded.iface, family=excluded.family,
			duration_sec=excluded.duration_sec`,
		rule.Backend, rule.Identity, rule.Protocol, rule.Port, rule.Action, "portpilot:"+rule.Identity,
		rule.Status, rule.Source, rule.Iface, rule.Family, rule.DurationSec)
	return err
}

// SetStatus changes active or disabled without deleting the spec.
func (r *Rules) SetStatus(ctx context.Context, id, status string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE firewall_rules SET status=? WHERE identity=?`, status, id)
	return err
}

// Delete removes a remembered spec.
func (r *Rules) Delete(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM firewall_rules WHERE identity=?`, id)
	return err
}

// List returns remembered specs, newest identity last.
func (r *Rules) List(ctx context.Context) ([]OwnedRule, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT identity, backend, proto, port, action, source, iface, family, duration_sec, status
		FROM firewall_rules ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OwnedRule
	for rows.Next() {
		var rule OwnedRule
		if err := rows.Scan(&rule.Identity, &rule.Backend, &rule.Protocol, &rule.Port, &rule.Action, &rule.Source, &rule.Iface, &rule.Family, &rule.DurationSec, &rule.Status); err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, rows.Err()
}
