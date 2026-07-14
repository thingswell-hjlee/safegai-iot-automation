# SafeGAI MVP 8+2주 최단기 개발계획 v3.0

## 목표
- Week 8: 테스트베드 통과 Release Candidate
- Week 10: 첫 현장 Pilot Release
- Gateway는 특정 SBC가 아닌 `ipc-lite-amd64-v1` 프로파일로 출시

## Week 0: 3일 준비
### D1
- Monorepo, Claude Settings, CI Skeleton
- Ubuntu 24.04 LTS amd64 Autoinstall 초안
- Hardware Profile Schema와 `ipc-lite-amd64-v1`
- Reference IPC·Alternate IPC 초기 설치
- Camera API Sample 수집
### T1
- 사용자/운영자 Workflow와 Acceptance Case
### T2
- IPC 2대, Testbed Wiring, Power/Network Fault Injection 준비
- BIOS Power Restore와 Dual NIC 확인

## Week 1: Scope/API/Hardware Spike
### D1
- Camera 1종 Event/Count/Health Spike
- Common Event Schema
- Gateway Skeleton, SQLite, Simulator
- Native amd64 Build와 `.deb` Skeleton
### T1
- 3개 Role Mode Screen Test Case
### T2
- Camera Zone/Count Accuracy Baseline
- Reference/Alternate IPC Qualification 1차
### Gate
- Camera API, Hardware Profile, Reference/Alternate IPC 확정

## Week 2: Occupancy Vertical Slice
### D1
- Camera Adapter
- Occupancy State Machine
- Local Event Store
- Basic Local Status UI
- Hardware/SSD/NIC Health Collector
### T1
- Event List/Status Workflow
### T2
- Enter/Exit/Count/Stale/Offline Tests
- 두 IPC 동일 결과 확인

## Week 3: Machine/I-O Safety Slice
### D1
- DI Equipment State
- Fixed Safety Rules
- DO Alarm/Stop Request/Audit
### T1
- Warning and Acknowledgement Scenarios
### T2
- Modbus I/O, Latency, Duplicate and Timeout HIL
### Gate
- R3 First Acceptance

## Week 4: Role-based Local Product
### D1
- USER/OPERATOR/MAINTAINER UI
- 1/2/4 Live Streams via MediaMTX
- Work-window and Restart Interlock
### T1
- Role/Permission/UX Regression
### T2
- 4-stream CPU/RAM/Thermal 8h Test on Reference IPC
- Alternate IPC 4-stream Smoke Test

## Week 5: AWS Minimal Cloud
### D1
- IoT Core/CDK
- Ingest/Admin API
- DynamoDB/S3/Cognito/SNS
- Cloud Operator UI
- Gateway Hardware Profile and Version Inventory
### T1
- Cloud Event/Status/Ack/Report
### T2
- WAN Outage, Queued Replay, Duplicate Test

## Week 6: Packaging/Recovery
### D1
- Signed amd64 `.deb`
- Ubuntu Autoinstall Finalization
- Backup/Restore/Update/Rollback
- Diagnostics and Health
- Release Workflows
### T1
- Maintainer Workflow and Manuals
### T2
- Power Loss, Disk Pressure, NIC Swap, Recovery Tests
- Reference -> Alternate IPC Migration Test

## Week 7: Full Verification
### D1
- Defect Correction Only
- Performance Optimization
### T1
- Full Functional Regression
### T2
- 72h Cloud Outage
- 20 Power Cycles
- Event Burst and 100k DB
- Hardware Qualification Final
### Gate
- No Open P0/R3 Defect

## Week 8: RC and Testbed Acceptance
### D1
- `v0.5.0-rc` amd64 Package
- SBOM/Signature/Rollback
- Qualified Model List와 OS Image ID
### T1
- Functional Sign-off
### T2
- HIL/Performance/Hardware Sign-off
### Output
- Field-installable RC

## Week 9: Pilot Installation
- Network, Camera Zones, I/O Mapping
- Reference IPC 설치
- Alternate IPC Recovery 준비
- Operator/Maintainer Training
- First-site Calibration
- Daily Defect Triage

## Week 10: Pilot Release
- Critical Fixes
- Site Acceptance
- `v0.9.0-pilot.1`
- 30-day Monitoring Start

## 일정 보호규칙
- Week 1 이후 신규 기능은 Backlog로 이동
- P0 결함과 현장 설치 Blocker만 현재 Sprint에 추가
- Generic Rule Editor, Cloud Live Video, OTA Automation, Predictive AI는 개발하지 않음
- 세 번째 IPC 모델 지원은 Pilot 이후로 이동
- Vendor-specific Driver나 GPIO 요구가 발생하면 해당 모델을 탈락시키는 것을 우선 검토
