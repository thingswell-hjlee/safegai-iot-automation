# AWS Infrastructure Instructions

## Scope
AWS CDK v2 TypeScript for dev and pilot environments.

## Services
IoT Core, named shadows, two Lambda functions, two DynamoDB tables, S3, Cognito, API Gateway HTTP API, CloudFront, SNS, CloudWatch, Budgets.

## Security
- GitHub Actions uses OIDC, not long-lived AWS keys.
- Gateway certificates are restricted to their own topic namespace.
- Pilot deploy requires protected environment approval.
- No public S3 bucket.
- No cloud machine-control API, topic, or shadow field.
- Cloud-to-device settings are a non-safety allowlist only.

## Automation boundary
Claude and Kiro may run `cdk synth` and `cdk diff`. They must not run deploy or destroy without explicit human approval.

## Required tests
- CDK assertions
- IAM policy checks
- topic policy tests
- DynamoDB key/query tests
- image lifecycle tests
- API authorization tests
- cost/budget alarm existence
