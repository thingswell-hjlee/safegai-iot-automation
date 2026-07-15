# Database Migrations

Migrations are embedded in the gateway binary at:
`services/gateway-server/internal/storage/sqlite/sqlite.go`

The gateway automatically applies pending migrations on startup.
Schema version is tracked in the `schema_migrations` table.

## Current Schema Version: 11

Tables:
- events - Safety event store with idempotency
- occupancy_states - Zone occupancy state tracking
- equipment_states - Equipment state tracking
- safety_decisions - Safety rule decision log
- actuation_results - Output command results
- audit_logs - Audit trail
- cloud_outbox - Cloud sync queue
- config_versions - Versioned configuration
- users - Local user accounts
- idempotency_keys - Event dedup tracking
- boot_records - Restart recovery

## Safety Guards

- **Idempotency**: Duplicate event IDs are rejected
- **Event Ordering**: Out-of-order events (by sequence number per device) are rejected
- **Stale Event**: Events older than 60s are rejected
- **Output Replay**: Boot records prevent replaying commands from previous sessions
- **Duplicate Output**: Actuation dedup prevents duplicate commands

## WAL Mode

SQLite is configured in WAL (Write-Ahead Logging) mode for:
- Concurrent read access during writes
- Crash recovery
- Better performance for the read-heavy workload
