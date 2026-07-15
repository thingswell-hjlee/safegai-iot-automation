# AWS Simulation Operation Runbook

## Daily Operations

### Start Environment
```bash
gh workflow run aws-sim-start.yml
```

### Stop Environment (Cost Saving)
```bash
gh workflow run aws-sim-stop.yml
```

Auto-stop runs daily at 10PM UTC via schedule.

### Check Status
```bash
aws ec2 describe-instances \
  --filters "Name=tag:Project,Values=safegai-sim" \
  --query 'Reservations[].Instances[].{Id:InstanceId,State:State.Name}'
```

## Monitoring

### CloudWatch Dashboard
Navigate to: SafeGAI-Sim-sim-dev dashboard in ap-northeast-2

### Key Metrics
- EC2 CPU Utilization (alarm at 80%)
- Network In/Out
- Custom: SafeGAI/Sim namespace

### Logs
```bash
aws ssm start-session --target <instance-id>
# Then on instance:
journalctl -u safegai-edge -f
journalctl -u safegai-camera-sim -f
```

## Running Tests

### Functional Test (S01-S14)
```bash
gh workflow run aws-sim-functional-test.yml -f scenarios=ALL
```

### Load Test
```bash
gh workflow run aws-sim-load-test.yml -f duration=5m -f concurrency=10
```

## Troubleshooting

### Instance won't start
1. Check EC2 console for instance state
2. Verify no capacity issues in the AZ
3. Check IAM role is valid

### Gateway unhealthy
1. SSM into instance
2. `systemctl status safegai-edge`
3. `journalctl -u safegai-edge --since "5 min ago"`
4. Check disk space: `df -h`
5. Check SQLite: `sqlite3 /var/lib/safegai/gateway.db "PRAGMA integrity_check;"`

### Simulator not generating events
1. `systemctl status safegai-camera-sim`
2. `curl http://localhost:9001/health`
3. Restart: `systemctl restart safegai-camera-sim`

### High CPU
1. Check which process: `top -b -n1`
2. If gateway: check event rate, possible infinite loop
3. If simulator: reduce event interval in environment config

## Emergency Procedures

### Immediate Stop All
```bash
aws ec2 stop-instances --instance-ids <instance-id>
```

### Full Teardown
```bash
gh workflow run aws-sim-destroy.yml -f confirm=DESTROY
```

## Cost Management

| Action | Saves |
|--------|-------|
| Auto-stop (nightly) | ~67% of EC2 cost |
| Manual stop when not testing | 100% of EC2 cost |
| Destroy environment | 100% of all costs |
| S3 lifecycle (7-day expiry) | Prevents storage growth |
| DynamoDB on-demand | No idle cost |
