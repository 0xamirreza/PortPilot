ALTER TABLE firewall_rules ADD COLUMN status TEXT NOT NULL DEFAULT 'active';
ALTER TABLE firewall_rules ADD COLUMN source TEXT NOT NULL DEFAULT '';
ALTER TABLE firewall_rules ADD COLUMN iface TEXT NOT NULL DEFAULT '';
ALTER TABLE firewall_rules ADD COLUMN family TEXT NOT NULL DEFAULT '';
ALTER TABLE firewall_rules ADD COLUMN duration_sec INTEGER NOT NULL DEFAULT 0;

CREATE UNIQUE INDEX IF NOT EXISTS firewall_rules_identity ON firewall_rules (identity);
