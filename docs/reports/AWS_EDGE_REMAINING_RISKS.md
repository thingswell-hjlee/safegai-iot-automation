# AWS Edge Remaining Risks

## Risk Register

### RISK-001: Safety Rule Validation with Real Hardware
- **Severity**: HIGH
- **Probability**: Medium
- **Impact**: Safety failure in production
- **Description**: Safety rules validated only with simulators; real hardware
  timing and behavior may differ from simulation
- **Mitigation**: Lab testing phase with real cameras and PLCs
- **Owner**: Safety Engineer
- **Status**: OPEN

### RISK-002: Output Response Time SLA
- **Severity**: HIGH
- **Probability**: Medium
- **Impact**: Stop request delivered too late
- **Description**: Output command delivery time not measured with real Modbus
  devices; PLC response time unknown
- **Mitigation**: Measure end-to-end latency in lab setup
- **Owner**: Controls Engineer
- **Status**: OPEN

### RISK-003: Camera API Compatibility
- **Severity**: MEDIUM
- **Probability**: Medium
- **Impact**: Camera events not received in production
- **Description**: Simulator generates ideal events; real cameras may have
  different API formats, timing, or error conditions
- **Mitigation**: Implement vendor-specific adapter; test with real cameras
- **Owner**: Integration Engineer
- **Status**: OPEN

### RISK-004: Network Reliability
- **Severity**: MEDIUM
- **Probability**: High
- **Impact**: Event loss during network interruption
- **Description**: Factory networks may have intermittent connectivity
  issues not present in AWS simulation
- **Mitigation**: Event buffering in gateway (already implemented via outbox)
- **Owner**: Network Engineer
- **Status**: MITIGATED (by design)

### RISK-005: SQLite Performance at Scale
- **Severity**: MEDIUM
- **Probability**: Low
- **Impact**: Performance degradation over time
- **Description**: SQLite write performance may degrade with large
  event history if WAL checkpointing is delayed
- **Mitigation**: Configure auto-checkpoint; periodic maintenance script
- **Owner**: Platform Engineer
- **Status**: PARTIALLY MITIGATED

### RISK-006: No Independent Safety Review
- **Severity**: HIGH
- **Probability**: N/A (compliance)
- **Impact**: Cannot deploy to production
- **Description**: Safety rules have not been independently reviewed
  by a certified safety engineer
- **Mitigation**: Engage third-party safety assessor
- **Owner**: Safety Manager
- **Status**: OPEN

### RISK-007: Upgrade Path in Production
- **Severity**: MEDIUM
- **Probability**: Medium
- **Impact**: Downtime during upgrade
- **Description**: Upgrade scripts tested in sim only; production may have
  different state, longer running DB, more data
- **Mitigation**: Test upgrade on copy of production data
- **Owner**: Operations Engineer
- **Status**: OPEN

### RISK-008: Single Point of Failure
- **Severity**: MEDIUM
- **Probability**: Low
- **Impact**: Complete safety system failure
- **Description**: Single gateway instance with no redundancy;
  hardware failure stops all monitoring
- **Mitigation**: Hardware watchdog; PLC safety relay as independent backup
- **Owner**: Safety Engineer
- **Status**: ACKNOWLEDGED (by design - PLC is independent backup)

## Risk Trend

| Sprint | Open HIGH | Open MEDIUM | Closed |
|--------|-----------|-------------|--------|
| Current | 3 | 4 | 0 |
| Target (post-lab) | 1 | 2 | 4 |
| Target (pre-pilot) | 0 | 1 | 6 |

## Acceptance Criteria for Risk Closure

Each risk can only be closed with:
1. Documented test evidence
2. Human review and approval
3. Acceptance by Safety Officer (for safety risks)
4. Updated in HUMAN_APPROVAL_REQUIRED.md
