# AWS SIM OIDC One-Time Setup

## Overview

This runbook documents the one-time setup required to enable GitHub Actions
to deploy the SafeGAI simulation environment using OIDC federation
(no long-lived AWS credentials).

## Prerequisites

- AWS CLI configured with admin access to the target account
- Target AWS Account ID
- GitHub repository: `thingswell-hjlee/safegai-iot-automation`

## Steps

### 1. Deploy OIDC Provider

```bash
aws cloudformation deploy \
  --template-file infra/aws/bootstrap/github-oidc-provider.yaml \
  --stack-name safegai-github-oidc-provider \
  --region ap-northeast-2 \
  --capabilities CAPABILITY_IAM
```

This creates the GitHub OIDC identity provider in your AWS account.
Only needed once per account.

### 2. Deploy Simulation Deploy Role

```bash
aws cloudformation deploy \
  --template-file infra/aws/bootstrap/github-oidc-sim-role.yaml \
  --stack-name safegai-sim-github-role \
  --region ap-northeast-2 \
  --capabilities CAPABILITY_NAMED_IAM \
  --parameter-overrides \
    GitHubOrg=thingswell-hjlee \
    GitHubRepo=safegai-iot-automation
```

### 3. Get the Role ARN

```bash
aws cloudformation describe-stacks \
  --stack-name safegai-sim-github-role \
  --query 'Stacks[0].Outputs[?OutputKey==`RoleArn`].OutputValue' \
  --output text
```

### 4. Configure GitHub Repository Secret

In the GitHub repository settings:
1. Go to Settings > Secrets and variables > Actions
2. Create a new repository secret:
   - Name: `AWS_SIM_DEPLOY_ROLE_ARN`
   - Value: (the ARN from step 3)

### 5. CDK Bootstrap (if first CDK deployment)

```bash
npx cdk bootstrap aws://<ACCOUNT_ID>/ap-northeast-2 \
  --trust <ACCOUNT_ID> \
  --cloudformation-execution-policies arn:aws:iam::aws:policy/AdministratorAccess
```

## Verification

Run the CI workflow to verify OIDC federation works:

```bash
gh workflow run aws-sim-ci.yml
```

## Security Considerations

- The OIDC trust is scoped to this specific repository
- Branch restrictions limit which branches can assume the role
- The deploy policy uses least-privilege for sim resources only
- No long-lived credentials are stored in GitHub secrets
- Session duration is limited to 1 hour

## Troubleshooting

### "Not authorized to perform sts:AssumeRoleWithWebIdentity"

- Verify the OIDC provider thumbprint is current
- Check the trust policy subject claim matches the repository
- Ensure the workflow has `id-token: write` permission

### CDK deployment fails with permission errors

- Review the `sim-deploy-policy.json` for missing actions
- Check CloudTrail for the specific denied action
- Add the required permission and redeploy the role stack

## Cleanup

To remove the OIDC setup:

```bash
aws cloudformation delete-stack --stack-name safegai-sim-github-role
aws cloudformation delete-stack --stack-name safegai-github-oidc-provider
```
