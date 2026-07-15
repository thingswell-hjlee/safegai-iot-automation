# Human Approval Required

## Purpose

This document lists all items that require human review and approval
before the SafeGAI gateway can be deployed to a pilot or production
environment. These items cannot be automated or self-certified.

## Approval Gates

### Gate 1: Safety Rule Review

| Item | Reviewer | Status | Date | Signature |
|------|----------|--------|------|-----------|
| R-01: Person detection -> Warning | Safety Engineer | PENDING | - | - |
| R-02: Vacancy confirmation -> Safe | Safety Engineer | PENDING | - | - |
| R-03: E-stop -> Immediate stop | Safety Engineer | PENDING | - | - |
| R-04: Sensor breach -> Alarm | Safety Engineer | PENDING | - | - |
| R-05: Comm loss -> Safe-side default | Safety Engineer | PENDING | - | - |
| Overall safety architecture | Safety Manager | PENDING | - | - |

### Gate 2: Hardware Integration Validation

| Item | Reviewer | Status | Date | Signature |
|------|----------|--------|------|-----------|
| Camera event reception (real camera) | Integration Eng. | PENDING | - | - |
| Modbus communication (real PLC) | Controls Eng. | PENDING | - | - |
| Output actuation (real devices) | Controls Eng. | PENDING | - | - |
| End-to-end response time measured | Safety Engineer | PENDING | - | - |
| Failure mode behavior verified | Safety Engineer | PENDING | - | - |

### Gate 3: Operational Readiness

| Item | Reviewer | Status | Date | Signature |
|------|----------|--------|------|-----------|
| Rollback procedure tested | Operations | PENDING | - | - |
| Backup/restore verified | Operations | PENDING | - | - |
| Monitoring and alerting confirmed | Operations | PENDING | - | - |
| On-call procedures documented | Operations | PENDING | - | - |
| Incident response plan | Operations Mgr. | PENDING | - | - |

### Gate 4: Security Clearance

| Item | Reviewer | Status | Date | Signature |
|------|----------|--------|------|-----------|
| Security architecture review | Security Eng. | PENDING | - | - |
| Credential management verified | Security Eng. | PENDING | - | - |
| Network security assessment | Security Eng. | PENDING | - | - |

### Gate 5: Management Approval

| Item | Reviewer | Status | Date | Signature |
|------|----------|--------|------|-----------|
| Risk register accepted | Project Manager | PENDING | - | - |
| Cost approved | Finance | PENDING | - | - |
| Pilot deployment approved | Ops Manager | PENDING | - | - |
| Production deployment approved | Safety Manager + Ops Manager | PENDING | - | - |

## Approval Process

1. Technical team completes implementation and testing
2. Evidence collected in `evidence/aws-edge-ready/`
3. Reviewers examine code, tests, and evidence
4. Each reviewer signs off on their gate items
5. All gates must be cleared before proceeding to next phase
6. This document is updated with approval dates and signatures

## Important Notes

- **Safety rules are fixed and cannot be modified at runtime**
- **No automated system can approve safety-critical changes**
- **Each approval must reference specific test evidence**
- **Approvals expire after 6 months and must be renewed**

## Contact

- Safety Manager: (to be assigned)
- Operations Manager: (to be assigned)
- Security Engineer: (to be assigned)
- Project Manager: (to be assigned)
