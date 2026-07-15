# Evidence Directory

This directory stores evidence artifacts for the AWS-First Edge-Ready release gates.

## Structure

```
evidence/aws-edge-ready/
  performance/     Load and soak test results
  functional/      E2E scenario execution results
  portability/     Parity verification evidence
  security/        Security review findings
```

## Evidence Collection

Evidence is collected by:
1. GitHub Actions workflows (uploaded as artifacts)
2. Manual test execution (stored here)
3. Human review sessions (signed documents)

## Retention

Evidence should be retained for the lifetime of the release plus 12 months.
