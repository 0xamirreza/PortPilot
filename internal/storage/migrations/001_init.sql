CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE TABLE action_history (
    id INTEGER PRIMARY KEY,
    recorded_at TEXT NOT NULL,
    action TEXT NOT NULL,
    backend TEXT NOT NULL DEFAULT '',
    protocol TEXT NOT NULL DEFAULT '',
    port INTEGER NOT NULL DEFAULT 0,
    target TEXT NOT NULL DEFAULT '',
    duration TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL,
    detail TEXT NOT NULL DEFAULT '',
    user_name TEXT NOT NULL DEFAULT ''
);

CREATE INDEX action_history_recorded_at ON action_history (recorded_at DESC);

CREATE TABLE application_state (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

-- Reserved for later phases. Kernel firewall state is not copied here.
CREATE TABLE firewall_rules (
    id INTEGER PRIMARY KEY,
    backend TEXT NOT NULL,
    identity TEXT NOT NULL,
    proto TEXT NOT NULL,
    port INTEGER NOT NULL,
    action TEXT NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE scheduled_rules (
    id INTEGER PRIMARY KEY,
    rule_id INTEGER NOT NULL REFERENCES firewall_rules (id) ON DELETE CASCADE,
    expires_at TEXT NOT NULL,
    mechanism TEXT NOT NULL,
    status TEXT NOT NULL
);

CREATE TABLE process_snapshots (
    id INTEGER PRIMARY KEY,
    captured_at TEXT NOT NULL,
    pid INTEGER NOT NULL,
    start_ticks INTEGER NOT NULL,
    name TEXT NOT NULL,
    exe TEXT NOT NULL DEFAULT ''
);

CREATE TABLE network_interfaces (
    id INTEGER PRIMARY KEY,
    captured_at TEXT NOT NULL,
    name TEXT NOT NULL,
    ifindex INTEGER NOT NULL,
    mtu INTEGER NOT NULL DEFAULT 0
);
