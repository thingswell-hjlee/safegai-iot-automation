# AWS Simulation Destroy Runbook

## Purpose

Completely remove all AWS simulation resources to stop all costs.

## When to Destroy

- End of development sprint
- Environment no longer needed
- Cost reduction required
- Before creating a fresh environment

## Pre-Destruction Checklist

- [ ] No active tests running
- [ ] No pending data to export
- [ ] Team notified of destruction
- [ ] Backup any needed evidence/data from S3

## Destruction via GitHub Actions

```bash
gh workflow run aws-sim-destroy.yml -f confirm=DESTROY
```

This requires typing "DESTROY" to confirm. Uses the `aws-sim-destroy` environment
which may require additional approval.

## Manual Destruction

```bash
cd infra/sim
npm ci
npx cdk destroy --all --force
```

## Post-Destruction Verification

```bash
# Check no stacks remain
aws cloudformation list-stacks \
  --query 'StackSummaries[?contains(StackName, `SafeGAI-Sim`) && StackStatus!=`DELETE_COMPLETE`]'

# Check no EC2 instances
aws ec2 describe-instances \
  --filters "Name=tag:Project,Values=safegai-sim" \
  --query 'Reservations[].Instances[?State.Name!=`terminated`].InstanceId'

# Check no orphaned S3 buckets
aws s3 ls | grep safegai

# Check no DynamoDB tables
aws dynamodb list-tables --query 'TableNames[?contains(@, `safegai`)]'
```

## Resources Destroyed

1. EC2 instance (stopped first, then terminated)
2. VPC and all networking components
3. Security groups
4. IAM roles and instance profiles
5. DynamoDB table (data deleted)
6. S3 buckets (objects deleted, then bucket)
7. IoT Core thing and policies
8. API Gateway
9. Lambda functions
10. Cognito user pool
11. CloudFront distribution
12. CloudWatch dashboard and alarms
13. EventBridge rules

## What is NOT Destroyed

- GitHub OIDC provider (shared, account-level)
- GitHub deploy role (leave for future deployments)
- CDK bootstrap stack (shared, account-level)
- GitHub repository secrets

## Re-creation

To recreate the environment after destruction:
```bash
gh workflow run aws-sim-deploy.yml -f environment=sim-dev -f action=deploy
```
