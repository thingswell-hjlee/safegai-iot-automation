# AWS-First Edge-Ready Gap Analysis

## Purpose

This document identifies gaps between the current AWS simulation implementation
and the production-ready local edge gateway deployment.

## Gap Categories

### Category A: Code-Complete but Untested in Real Hardware

| ID | Gap | Risk | Remediation |
|----|-----|------|-------------|
| A1 | Camera adapter tested with simulator only | Medium | Lab test with real cameras |
| A2 | Modbus adapter tested with TCP sim only | Medium | Lab test with real PLC |
| A3 | Output adapter tested with sim only | High | Bench test with real relays/lights |
| A4 | Media proxy (RTSP) not tested end-to-end | Low | Lab test with MediaMTX + camera |

### Category B: Implementation Incomplete

| ID | Gap | Risk | Remediation |
|----|-----|------|-------------|
| B1 | Cloud IoT Core adapter not integrated | Low | Implement AwsIoTCloudAdapter |
| B2 | Real camera vendor adapter not implemented | Medium | Implement per vendor API |
| B3 | Notification adapter (SNS) not connected | Low | Implement when cloud integration needed |
| B4 | DynamoDB event query API (Lambda) is placeholder | Low | Implement proper query logic |

### Category C: Security Hardening

| ID | Gap | Risk | Remediation |
|----|-----|------|-------------|
| C1 | TLS not enforced on local API | Medium | Add TLS termination config |
| C2 | Session secret from env variable | Medium | Use secrets management |
| C3 | RBAC permissions not fully granular | Low | Add per-zone permissions |
| C4 | Audit log integrity (no signing) | Low | Add HMAC signature chain |

### Category D: Operations

| ID | Gap | Risk | Remediation |
|----|-----|------|-------------|
| D1 | Automated rollback not tested | Medium | Run rollback scenario in lab |
| D2 | Backup/restore for SQLite not tested under load | Medium | Soak test with backup |
| D3 | Log rotation under high volume not verified | Low | Extended soak test |
| D4 | Upgrade path only tested in simulation | Medium | Lab upgrade test |

### Category E: Safety Certification

| ID | Gap | Risk | Remediation |
|----|-----|------|-------------|
| E1 | Safety rules not independently reviewed | High | External safety review |
| E2 | Output response time SLA not measured | High | Lab measurement with real hardware |
| E3 | Failure mode analysis not completed | High | FMEA document needed |
| E4 | Human-in-the-loop approval evidence missing | High | Requires human sign-off |

## Priority Matrix

```
             High Risk
                |
    E1,E2  |  A3,E3
    E4     |
  ---------+---------
           |
    B2     |  A1,A2,D1
    C1,C2  |  D2,D4
           |
             Low Risk
   Effort: High    Low
```

## Recommended Remediation Order

1. **Immediate (before pilot)**: E1, E2, E3, E4, A3
2. **Short-term (pilot phase)**: A1, A2, C1, C2, D1, D4
3. **Medium-term (production)**: B1, B2, B3, B4, C3, C4, D2, D3
4. **Long-term (scaling)**: A4

## Evidence Requirements for Each Gap

Each gap closure requires:
- Test execution evidence (screenshots, logs)
- Human approval signature
- Timestamp of verification
- Commit reference to the fix

Evidence stored in: `evidence/aws-edge-ready/`
