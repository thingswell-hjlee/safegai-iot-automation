# AWS Simulation Deploy Runbook

## Purpose

Guide for deploying the SafeGAI simulation environment to AWS.

## Prerequisites

- GitHub OIDC configured (see AWS_SIM_OIDC_ONE_TIME_SETUP.md)
- CDK bootstrapped in target account/region
- `AWS_SIM_DEPLOY_ROLE_ARN` secret configured in GitHub

## Deployment via GitHub Actions (Recommended)

### 1. Deploy

```bash
gh workflow run aws-sim-deploy.yml \
  --ref feature/aws-first-edge-ready \
  -f environment=sim-dev \
  -f action=deploy
```

### 2. Verify Deployment

```bash
# Check workflow status
gh run list --workflow=aws-sim-deploy.yml --limit=1

# Once deployed, verify via SSM
aws ssm start-session --target <instance-id>
curl http://localhost:8080/health/ready
```

### 3. Check Stack Status

```bash
aws cloudformation describe-stacks \
  --query 'Stacks[?contains(StackName, `SafeGAI-Sim`)].{Name:StackName,Status:StackStatus}'
```

## Manual CDK Deployment

```bash
cd infra/sim
npm ci
npx cdk diff --all
npx cdk deploy --all --require-approval never
```

## Post-Deployment Checklist

- [ ] EC2 instance running (check EC2 console)
- [ ] Gateway health OK (`/health/ready` returns 200)
- [ ] All simulators running (check systemd)
- [ ] IoT Core thing registered
- [ ] DynamoDB table created
- [ ] S3 bucket created
- [ ] API Gateway accessible
- [ ] CloudFront distribution deployed
- [ ] CloudWatch dashboard visible
- [ ] Auto-stop rule active

## Stack Order

Stacks deploy in dependency order:
1. Network (VPC, subnets, SG)
2. Identity (IAM roles)
3. Data (DynamoDB, S3)
4. IoT (Thing, rules)
5. Gateway (EC2 instance)
6. API (API Gateway, Lambda)
7. Frontend (S3, CloudFront)
8. Observability (Dashboard, auto-stop)

## Troubleshooting

### CDK deployment hangs on EC2 instance
- User-data script may be failing
- Check `/var/log/safegai-user-data.log` via SSM

### Gateway not responding
- SSH via SSM: `aws ssm start-session --target <instance-id>`
- Check: `systemctl status safegai-edge`
- Logs: `journalctl -u safegai-edge -n 50`

### Stack rollback
- Check CloudFormation events for the error
- Fix the issue and redeploy
- Use `cdk diff` to preview changes before deploy
