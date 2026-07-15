# AWS Edge Cost Report

## Monthly Cost Estimate (Simulation Environment)

### With Auto-Stop (8 hours/day operation)

| Service | Unit | Quantity | Unit Cost | Monthly |
|---------|------|----------|-----------|---------|
| EC2 t3.medium | hours | 240 | $0.0464/hr | $11.14 |
| EBS gp3 30GB | GB-month | 30 | $0.08/GB | $2.40 |
| DynamoDB On-Demand | RCU/WCU | low | pay-per-use | $3.00 |
| S3 Standard | GB | 5 | $0.023/GB | $0.12 |
| S3 PUT requests | 1000 req | 100 | $0.005 | $0.50 |
| IoT Core messages | 1M msg | 0.1 | $1.00 | $0.10 |
| CloudFront | GB transfer | 1 | $0.085 | $0.09 |
| API Gateway | 1M requests | 0.01 | $3.50 | $0.04 |
| Lambda | 1M req + GB-s | minimal | - | $0.50 |
| CloudWatch | dashboards | 1 | $3.00 | $3.00 |
| NAT Gateway | - | 0 | - | $0.00 |
| **Total** | | | | **~$21/month** |

### 24/7 Operation (No Auto-Stop)

| Service | Change | Monthly |
|---------|--------|---------|
| EC2 t3.medium | 720 hours | $33.41 |
| Other services | Same | $9.75 |
| **Total** | | **~$43/month** |

## Cost Optimization Measures

1. **No NAT Gateway**: Saves ~$32/month by using public subnets only
2. **Auto-Stop**: EventBridge stops instance nightly (saves 67% EC2 cost)
3. **DynamoDB On-Demand**: No provisioned capacity, pay only for usage
4. **S3 Lifecycle**: 7-day retention prevents storage growth
5. **Spot not used**: Stability preferred for testing
6. **Single AZ**: Sim doesn't need multi-AZ redundancy

## Cost Comparison: AWS Sim vs Local Lab

| Category | AWS Sim | Local Lab |
|----------|---------|-----------|
| Hardware | $0 | $500-2000 (one-time) |
| Monthly compute | $21 | $0 (electricity only) |
| Maintenance | Automated | Manual |
| Availability | On-demand | Always-on |
| Scaling | Easy | Hardware limited |
| 1-year TCO | $252 | $500-2000 |

## Destroy Cost: $0

When not needed, destroy the environment completely:
```bash
gh workflow run aws-sim-destroy.yml -f confirm=DESTROY
```

All resources are destroyed and costs stop immediately.

## Budget Alerts

Set up AWS Budget alarm:
- Warning at $30/month
- Critical at $50/month

```bash
aws budgets create-budget --account-id <ACCOUNT_ID> --budget '{
  "BudgetName": "SafeGAI-Sim-Monthly",
  "BudgetLimit": {"Amount": "50", "Unit": "USD"},
  "TimeUnit": "MONTHLY",
  "BudgetType": "COST",
  "CostFilters": {"TagKeyValue": ["user:Project$safegai-sim"]}
}'
```

## Conclusion

The simulation environment costs approximately $21/month with auto-stop
enabled. This is significantly less than maintaining a physical lab
environment while providing equivalent testing capability for the
gateway software.
