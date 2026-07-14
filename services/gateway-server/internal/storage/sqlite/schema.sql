-- SafeGAI Gateway SQLite Schema
-- All timestamps stored as ISO-8601 UTC strings.
-- WAL mode must be enabled at connection time: PRAGMA journal_mode=WAL;

PRAGMA journal_mode=WAL;
PRAGMA foreign_keys=ON;

-- Safety events from all sources (camera, device, zone engine, etc.)
CREATE TABLE IF NOT EXISTS events (
    id              TEXT PRIMARY KEY,
    correlation_id  TEXT NOT NULL,
    tenant_id       TEXT NOT NULL,
    site_id         TEXT NOT NULL,
    gateway_id      TEXT NOT NULL,
    camera_id       TEXT,
    zone_id         TEXT,
    severity        TEXT NOT NULL CHECK (severity IN ('INFO','WARNING','CRITICAL','ALARM')),
    occupancy_state TEXT NOT NULL CHECK (occupancy_state IN ('UNKNOWN','OCCUPIED','VACANT_PENDING','VACANT_CONFIRMED','STALE')),
    equipment_state TEXT NOT NULL CHECK (equipment_state IN ('RUNNING','STOPPED','FAULT','UNKNOWN')),
    actions         TEXT, -- JSON array of action strings
    detected_at     TEXT NOT NULL,
    received_at     TEXT NOT NULL,
    ack_by          TEXT,
    ack_at          TEXT,
    resolved_by     TEXT,
    resolved_at     TEXT,
    classification  TEXT,
    image_key       TEXT
);

CREATE INDEX IF NOT EXISTS idx_events_correlation ON events(correlation_id);
CREATE INDEX IF NOT EXISTS idx_events_zone ON events(zone_id);
CREATE INDEX IF NOT EXISTS idx_events_severity ON events(severity);
CREATE INDEX IF NOT EXISTS idx_events_detected_at ON events(detected_at);

-- Zone occupancy state tracking
CREATE TABLE IF NOT EXISTS occupancy_states (
    zone_id         TEXT PRIMARY KEY,
    state           TEXT NOT NULL CHECK (state IN ('UNKNOWN','OCCUPIED','VACANT_PENDING','VACANT_CONFIRMED','STALE')),
    previous_state  TEXT CHECK (previous_state IN ('UNKNOWN','OCCUPIED','VACANT_PENDING','VACANT_CONFIRMED','STALE')),
    person_count    INTEGER NOT NULL DEFAULT 0,
    changed_at      TEXT NOT NULL,
    event_id        TEXT REFERENCES events(id)
);

-- Equipment operational state tracking
CREATE TABLE IF NOT EXISTS equipment_states (
    equipment_id    TEXT PRIMARY KEY,
    state           TEXT NOT NULL CHECK (state IN ('RUNNING','STOPPED','FAULT','UNKNOWN')),
    quality         TEXT NOT NULL CHECK (quality IN ('GOOD','UNCERTAIN','BAD','STALE')),
    last_update     TEXT NOT NULL
);

-- Safety decision audit trail
CREATE TABLE IF NOT EXISTS safety_decisions (
    id              TEXT PRIMARY KEY,
    correlation_id  TEXT NOT NULL,
    rule            TEXT NOT NULL,
    zone_id         TEXT,
    equipment_id    TEXT,
    decision        TEXT NOT NULL,
    reason          TEXT NOT NULL,
    timestamp       TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_decisions_correlation ON safety_decisions(correlation_id);
CREATE INDEX IF NOT EXISTS idx_decisions_zone ON safety_decisions(zone_id);

-- Actuation command results
CREATE TABLE IF NOT EXISTS actuation_results (
    command_id      TEXT PRIMARY KEY,
    correlation_id  TEXT NOT NULL,
    command_type    TEXT NOT NULL,
    target          TEXT NOT NULL,
    status          TEXT NOT NULL CHECK (status IN ('PENDING','SENT','CONFIRMED','FAILED','TIMEOUT')),
    executed_at     TEXT NOT NULL,
    latency_ms      INTEGER,
    error           TEXT
);

CREATE INDEX IF NOT EXISTS idx_actuation_correlation ON actuation_results(correlation_id);

-- Audit log for all user and system actions
CREATE TABLE IF NOT EXISTS audit_logs (
    id              TEXT PRIMARY KEY,
    timestamp       TEXT NOT NULL,
    actor           TEXT NOT NULL,
    role            TEXT NOT NULL,
    action          TEXT NOT NULL,
    target          TEXT NOT NULL,
    detail          TEXT,
    ip              TEXT
);

CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_logs(actor);

-- Cloud outbox for reliable message delivery
CREATE TABLE IF NOT EXISTS cloud_outbox (
    id              TEXT PRIMARY KEY,
    event_id        TEXT NOT NULL REFERENCES events(id),
    payload         BLOB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','sent','failed')),
    created_at      TEXT NOT NULL,
    sent_at         TEXT,
    retry_count     INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON cloud_outbox(status);
CREATE INDEX IF NOT EXISTS idx_outbox_created ON cloud_outbox(created_at);

-- Configuration version history
CREATE TABLE IF NOT EXISTS config_versions (
    id              TEXT PRIMARY KEY,
    version         INTEGER NOT NULL UNIQUE,
    content         TEXT NOT NULL,
    created_at      TEXT NOT NULL,
    created_by      TEXT NOT NULL,
    active          INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_config_active ON config_versions(active);

-- Local user accounts with role-based access
CREATE TABLE IF NOT EXISTS users (
    id                      TEXT PRIMARY KEY,
    username                TEXT NOT NULL UNIQUE,
    password_hash           TEXT NOT NULL,
    role                    TEXT NOT NULL CHECK (role IN ('USER','OPERATOR','MAINTAINER')),
    created_at              TEXT NOT NULL,
    last_login              TEXT,
    force_password_change   INTEGER NOT NULL DEFAULT 1
);

CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
