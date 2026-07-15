# Local Gateway Rollback Runbook

## Purpose

Safely roll back the SafeGAI edge gateway to a previous version
if an upgrade causes issues.

## When to Rollback

- Health check fails after upgrade
- Safety rule behavior changed unexpectedly
- Performance degradation detected
- Hardware communication broken

## Automated Rollback

The upgrade script supports automatic rollback:

```bash
sudo /usr/local/bin/safegai-rollback.sh
```

This restores the previous binary and database backup.

## Manual Rollback Steps

### 1. Stop Current Service

```bash
sudo systemctl stop safegai-edge
```

### 2. Restore Previous Binary

```bash
# Previous version is kept in backup location
sudo cp /var/lib/safegai/backup/safegai-edge /usr/local/bin/safegai-edge
```

### 3. Restore Database (if needed)

```bash
# Only if schema migration occurred
sudo cp /var/lib/safegai/backup/gateway.db /var/lib/safegai/gateway.db
```

### 4. Restart Service

```bash
sudo systemctl start safegai-edge
```

### 5. Verify

```bash
curl -s http://localhost:8080/health/ready | jq .version
# Should show the previous version
```

## Rollback Checklist

- [ ] Service stopped cleanly
- [ ] Previous binary restored
- [ ] Database restored (if schema changed)
- [ ] Service restarts successfully
- [ ] Health check passes
- [ ] Safety rules still evaluate correctly
- [ ] Hardware communication verified
- [ ] Incident documented

## Prevention

Before upgrading:
1. Run `scripts/backup.sh` to create a restore point
2. Verify the new version in local-sim profile first
3. Test safety scenarios before switching to pilot
4. Keep the previous .deb package accessible

## Database Considerations

- Schema version is tracked in SQLite
- Forward migrations are automatic
- Reverse migrations are NOT automatic
- Always backup before upgrade
- If schema changed, must restore backup to rollback

## Emergency Contacts

- Incident channel: (team-specific)
- On-call: (team-specific)
- Safety officer: (required for safety-related rollbacks)
