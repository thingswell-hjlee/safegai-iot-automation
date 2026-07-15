# AWS Simulation Functional Test Report

## Summary

| Metric | Value |
|--------|-------|
| Execution Date | (Pending - requires deployed environment) |
| Environment | sim-dev |
| Total Scenarios | 14 |
| Executed | 0 |
| Passed | 0 |
| Failed | 0 |
| Skipped | 0 |
| Status | PENDING DEPLOYMENT |

## Scenario Results

| ID | Name | Status | Duration | Notes |
|----|------|--------|----------|-------|
| S01 | Person enters hazard zone | PENDING | - | - |
| S02 | Zone vacancy confirmed | PENDING | - | - |
| S03 | Emergency stop | PENDING | - | - |
| S04 | Sensor threshold breach | PENDING | - | - |
| S05 | Communication loss | PENDING | - | - |
| S06 | Equipment fault | PENDING | - | - |
| S07 | Multi-zone occupancy | PENDING | - | - |
| S08 | Restart interlock | PENDING | - | - |
| S09 | Network partition | PENDING | - | - |
| S10 | Modbus DI alarm | PENDING | - | - |
| S11 | Voice announcement | PENDING | - | - |
| S12 | Audit trail completeness | PENDING | - | - |
| S13 | Concurrent events | PENDING | - | - |
| S14 | Graceful shutdown | PENDING | - | - |

## How to Execute

```bash
# Deploy environment first
gh workflow run aws-sim-deploy.yml -f environment=sim-dev -f action=deploy

# Wait for deployment (~5 min)

# Run functional tests
gh workflow run aws-sim-functional-test.yml -f scenarios=ALL
```

## Test Environment

- Instance: t3.medium (sim-dev)
- Region: ap-northeast-2
- Gateway version: 0.1.0
- Simulators: All running (camera, sensor, equipment, output, modbus)

## Prerequisites for Execution

1. AWS simulation environment deployed and healthy
2. All simulator services running
3. Gateway responding to health checks
4. Scenario runner accessible via SSM

## Expected Completion

This report will be updated after the first successful deployment and
execution of the functional test workflow.
