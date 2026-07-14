# SafeGAI 초기 GitHub Backlog
## 8주 RC를 위한 Issue 구성

각 Issue는 1~2일 안에 완료할 수 있는 크기로 유지한다. 번호는 실제 GitHub 생성 순서에 따라 바뀔 수 있다.

---

# Epic E0: 저장소·도구·시험기반

## ISSUE-001 Repository Bootstrap

**가치:** 모든 변경과 증거를 재현 가능하게 관리한다.  
**Risk:** R0  
**Owner:** D1  
**Tester:** T1

Acceptance:

- Private Monorepo
- main Ruleset
- CODEOWNERS
- Issue/PR Template
- GitHub Project
- `make check-prereqs`, `make verify-fast`, `make verify`
- Secret Scan 0건

## ISSUE-002 Toolchain Version Pinning

**Risk:** R2

Acceptance:

- Go, Node, npm, AWS CLI, CDK Version Manifest
- CI와 개발 PC 동일 Major Version
- Gateway amd64 Build
- Version 변경 절차 문서화

## ISSUE-003 PR CI Skeleton

Acceptance:

- Markdown·JSON·YAML 검사
- Shellcheck
- Secret Scan
- Hardware Profile Schema
- Path 기반 Go/TS/CDK 검사
- Artifact 보관

## ISSUE-004 Testbed BOM and Wiring

**Owner:** T2  
**Risk:** R2

Acceptance:

- BOM
- 전원·Network·I/O Wiring
- Test Relay
- 생산설비 물리분리
- 전원·Network Fault 방법

## ISSUE-005 Acceptance Test Skeleton

**Owner:** T1

Acceptance:

- 사용자·운영자·유지보수자 업무목록
- 예상·금지 동작
- P0~P3 기준
- Evidence Template

---

# Epic E1: 카메라·계약

## ISSUE-010 Fisheye Camera API Spike

**Risk:** R2  
**Gate:** G1

Acceptance:

- Zone ID
- Occupied·Vacant 또는 Event Start·End
- Count 가능여부
- Enter·Exit
- Snapshot
- Health
- Reconnect
- 20회 반복결과
- Capability Matrix

## ISSUE-011 Camera Selection ADR

Acceptance:

- 기준모델·Firmware
- 사용 API
- 알려진 한계
- 대체모델
- Go/No-Go 근거

## ISSUE-012 Common Camera Event Schema

**Risk:** R2

Acceptance:

- JSON Schema
- Good/Bad Examples
- Timestamp·Sequence·Quality
- Snapshot Reference
- Contract Test

## ISSUE-013 Occupancy State Contract

**Risk:** R3  
**Tester:** T1+T2

Acceptance:

- 5개 상태 정의
- State Transition Table
- TTL·Stale Rule
- Invalid Transition
- Fail-safe Rule

## ISSUE-014 Safety Decision Contract

**Risk:** R3

Acceptance:

- Decision Types
- Inputs and Quality
- Actions
- Correlation ID
- No Automatic Restart

## ISSUE-015 Local API Contract

Acceptance:

- Health
- Current State
- Events
- ACK·Resolve·Classify
- Work Window
- Maintainer Diagnostics

## ISSUE-016 MQTT Topic and Cloud Event Contract

Acceptance:

- Topic Naming
- QoS
- Payload Limit
- Idempotency
- Image Transfer
- No Control Topic

---

# Epic E2: 시뮬레이터

## ISSUE-020 Camera Simulator

Acceptance:

- Occupied·Vacant
- Count 0~5
- Enter·Exit·Dwell
- Duplicate·Delay·Out-of-order
- Offline·Malformed
- Scriptable Scenarios

## ISSUE-021 Modbus I/O Simulator

Acceptance:

- Running·Stopped
- Restart Request
- DO Command
- Feedback
- Timeout·Exception·Offline

## ISSUE-022 Scenario Runner

Acceptance:

- YAML/JSON Scenario
- Relative Time
- Expected State·Action
- JUnit or JSON Result
- CI Artifact

---

# Epic E3: Gateway Core

## ISSUE-030 Gateway Process Skeleton

Acceptance:

- Config
- Logger
- Build Info
- Live/Ready Health
- Graceful Shutdown
- Hardware Profile Check

## ISSUE-031 SQLite WAL and Migration

Acceptance:

- Initial Tables
- WAL
- UTC
- Unique Event ID
- Migration Up/Down Policy
- Crash Recovery Test

## ISSUE-032 Event Normalizer

Acceptance:

- Manufacturer Payload Mapping
- Validation
- Reject Reason
- Duplicate Filter
- Raw Reference

## ISSUE-033 Occupancy State Machine

**Risk:** R3

Acceptance:

- Transition Tests 100%
- Timeout→STALE
- No Missing Data→VACANT
- Count Conflict→UNKNOWN
- Restart Recovery

## ISSUE-034 Equipment State Adapter

**Risk:** R3

Acceptance:

- DI/Modbus Running
- Restart Request
- TTL
- Offline→UNKNOWN
- Audit

## ISSUE-035 Fixed Safety Rule Template 1

**Risk:** R3

Acceptance:

- OCCUPIED + RUNNING
- Warning
- Stop Required
- Correlation
- Duplicate suppression

## ISSUE-036 Restart Interlock Template

**Risk:** R3

Acceptance:

- Restart Request + not VACANT_CONFIRMED
- Interlock Result
- No automatic restart
- UNKNOWN/STALE fail-safe

## ISSUE-037 Actuation Service

**Risk:** R3

Acceptance:

- Lamp·Buzzer·Stop Pulse
- Command ID
- Timeout
- Optional Feedback
- Retry Limit
- Boot does not replay Pulse

## ISSUE-038 Audit Log

Acceptance:

- Event·Decision·Action·User·Config
- UTC Timestamp
- Immutable Application API
- Export

## ISSUE-039 Cloud Outbox

**Risk:** R2

Acceptance:

- Transactional Insert
- Retry Backoff
- Idempotency
- Queue Metrics
- Dead Letter State

---

# Epic E4: Local Frontend and Roles

## ISSUE-040 Local Authentication and RBAC

**Risk:** R2

Acceptance:

- Argon2id
- Session Token
- User·Operator·Maintainer
- API-level Authorization
- Login Rate Limit
- Initial Password Change

## ISSUE-041 User Mode

Acceptance:

- Safety State
- Zone State
- Equipment State
- Active Warning
- Action Guide
- No ACK/Config

## ISSUE-042 Operator Mode

Acceptance:

- Event Queue
- Detail
- ACK·Resolve·Classify
- Work Window
- Report
- No Safety Mapping

## ISSUE-043 Maintainer Mode

**Risk:** R2

Acceptance:

- Camera Test
- Zone Mapping
- DI Live
- DO Test in TEST
- Health·Log
- Backup·Restore
- Local Network Only

## ISSUE-044 MediaMTX 1/2/4 View

Acceptance:

- H.264 Substream
- 4 View 8h
- Fullscreen 1080p
- Offline Placeholder
- No Transcoding

## ISSUE-045 Work Window and TEST State

**Risk:** R3

Acceptance:

- Approval
- Start/End
- Auto Expiry
- Audit
- TEST Exit Output Off

---

# Epic E5: AWS Minimal Cloud

## ISSUE-050 AWS CDK Baseline

Acceptance:

- Dev Stack
- Pilot Stack Parameter
- Tags
- Budget Alarm
- OIDC Role Skeleton
- `cdk synth`

## ISSUE-051 IoT Thing Provisioning

**Risk:** R2

Acceptance:

- Thing
- Certificate
- Least Privilege Policy
- Topic Restriction
- Rotation Procedure

## ISSUE-052 Ingest Lambda

Acceptance:

- Schema Validation
- Identity Validation
- Idempotent Event
- Gateway Last Seen
- Structured Log

## ISSUE-053 DynamoDB Event and Gateway Tables

Acceptance:

- Partition Keys
- Query Patterns
- TTL Policy
- Conditional Write
- Backup Policy

## ISSUE-054 S3 Image Ingest

Acceptance:

- Raw JPEG Limit
- Object Key
- Encryption
- Lifecycle
- Presigned Read

## ISSUE-055 SNS Notification

Acceptance:

- Critical Event
- Device Offline
- Dedup/Cooldown
- Delivery Log

## ISSUE-056 Admin API Lambda

Acceptance:

- Site State
- Event List/Detail
- ACK·Resolve·Classify
- Report
- No Machine Control

## ISSUE-057 Cognito and Cloud Operator UI

Acceptance:

- Login
- Role
- Event·Status
- Image
- ACK Sync
- Session Timeout

## ISSUE-058 Named Device Shadows

Acceptance:

- `health` reported-only
- `settings` allowlist
- No Safety Desired State
- Policy Separation

## ISSUE-059 GitHub OIDC Dev Deployment

Acceptance:

- No Long-lived AWS Key
- Branch/Environment subject restriction
- Dev Auto Deploy
- Pilot Manual Approval

---

# Epic E6: Provisioning and Release

## ISSUE-060 Ubuntu Autoinstall

**Risk:** R2

Acceptance:

- UEFI
- Disk Layout
- SSH Key
- Dual NIC
- Wi-Fi Off
- OS Image ID

## ISSUE-061 amd64 DEB Package

Acceptance:

- Install
- Upgrade
- Uninstall Policy
- systemd
- File Ownership
- Config Preserve

## ISSUE-062 Backup and Restore

**Risk:** R2

Acceptance:

- Config Version
- DB Backup
- Integrity Check
- Reference→Alternate IPC
- Audit

## ISSUE-063 Update and Rollback

**Risk:** R2

Acceptance:

- Signature
- Disk Check
- DB Backup
- Health Gate
- Automatic Rollback on Failure

## ISSUE-064 Diagnostics Bundle

Acceptance:

- Logs
- Version
- Hardware
- Health
- Redaction
- Export

## ISSUE-065 Release Workflow

Acceptance:

- SBOM
- Checksum
- Signature
- Manifest
- Evidence Links
- Protected Pilot Environment

---

# Epic E7: Verification and Pilot

## ISSUE-070 Functional Regression

**Owner:** T1

Acceptance:

- 3 Modes
- Permission
- Event Lifecycle
- Report
- No Open P0/P1

## ISSUE-071 HIL Latency

**Owner:** T2

Acceptance:

- Alarm p95 <= 1000ms
- DO p95 <= 500ms
- Evidence CSV
- Reference and Alternate IPC

## ISSUE-072 72h Offline and Replay

Acceptance:

- Local Safety continues
- Outbox retained
- Recovery no duplicate
- Queue Metrics

## ISSUE-073 Power Cycle 20

Acceptance:

- Auto Power On
- Safety Ready <= 3min
- No Old Pulse Replay
- DB Integrity

## ISSUE-074 100k Event DB and 50k Outbox

Acceptance:

- Query Response Target
- Disk Usage
- Replay
- No Data Corruption

## ISSUE-075 4 Stream 8h

Acceptance:

- CPU/RAM/Thermal
- No Stream Leak
- UI Recover

## ISSUE-076 Hardware Qualification Reference IPC

Acceptance:

- Full Matrix
- Evidence ID
- Qualified Model Entry

## ISSUE-077 Hardware Qualification Alternate IPC

Acceptance:

- Same Package
- Same Result
- Restore Test

## ISSUE-078 RC Release

Acceptance:

- T1/T2 Sign-off
- Release Assets
- Rollback
- Installation Guide

## ISSUE-079 Pilot Site Installation

Acceptance:

- Site Survey
- Zone Calibration
- Test Relay
- PLC Approval
- Operator Training
- Acceptance Record

## ISSUE-080 30-day Pilot Review

Acceptance:

- KPI
- Defects
- User Feedback
- v1.0 Go/No-Go
