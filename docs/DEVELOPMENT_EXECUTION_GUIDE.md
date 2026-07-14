# SafeGAI 개발 실행 가이드
## 1인 개발 + 2인 시험, 8주 RC + 2주 Pilot

문서 목적은 처음 개발을 시작하는 시점부터 현장 Pilot까지 무엇을, 어떤 순서로, 어떤 증거를 남기며 수행할지 정하는 것이다.

---

# 1. 개발 성공조건

MVP 성공은 기능 수가 아니라 다음 결과로 판단한다.

1. AI 어안카메라의 재실 이벤트가 안정적으로 Gateway에 수신된다.
2. 설비 운전상태와 재실상태를 결합해 고정형 안전조건을 판단한다.
3. 경고 및 PLC/Safety Relay 정지요청을 정해진 시간 안에 실행한다.
4. 카메라·인터넷·AWS가 장애여도 안전하지 않은 상태를 안전으로 오판하지 않는다.
5. 사용자·운영자·유지보수자의 권한과 화면이 분리된다.
6. 모든 사건·판단·출력·확인·설정변경을 추적할 수 있다.
7. 두 종류의 범용 amd64 IPC에서 같은 패키지가 동작한다.
8. 시험증거가 GitHub Issue, PR, Release에 연결된다.

---

# 2. 개발조직과 일일 운영

## 2.1 D1 개발자·PL

D1은 설계, Gateway, AWS, Frontend, 패키징을 담당한다. 그러나 R3 안전변경을 단독 승인하지 않는다.

매일 오전:

1. 전일 T1·T2 결함 검토
2. 당일 Issue 1개 선택
3. Acceptance Criteria와 Risk 등급 확인
4. Claude Code Plan 실행
5. 구현 범위 1~2일 이하로 제한

매일 오후:

1. `make verify-fast`
2. 기능별 수동 Smoke Test
3. PR Draft 생성
4. T1 또는 T2에 Nightly Build 전달
5. 미완료 작업과 위험 기록

## 2.2 T1 기능·UX 시험자

T1은 사용자·운영자 업무를 요구사항 기준으로 확인한다.

매일 수행:

- 전일 Build 설치 또는 웹 접속
- 사용자 모드의 이해성·오조작 확인
- 운영자 ACK·조치·종료 흐름 확인
- 권한우회 시험
- 결함 Issue 작성
- 회귀시험 결과 등록

T1은 “화면이 보인다”가 아니라 “현장 사용자가 올바르게 판단하고 잘못된 조작을 하지 않는다”를 검증한다.

## 2.3 T2 성능·HIL 시험자

T2는 실제 IPC, 카메라, Modbus I/O, 전원과 네트워크를 시험한다.

매일 수행:

- Nightly amd64 패키지 설치
- 카메라·I/O 연결 확인
- 지연·CPU·RAM·온도 측정
- 전원·통신·카메라 장애 주입
- 출력 피드백 확인
- HIL 증거 업로드

T2는 실제 생산설비에 연결하기 전에 Test Relay와 표시등으로 정지요청을 검증한다.

---

# 3. 단계별 Gate

| Gate | 통과조건 | 실패 시 조치 |
|---|---|---|
| G0 범위 | 제품범위·역할·안전경계 승인 | 신규 기능 제거 또는 Backlog 이동 |
| G1 카메라 | 재실·인원·진입·이탈·Health API 검증 | 카메라 교체 또는 범위 축소 |
| G2 계약 | Event·State·Safety Decision Schema 승인 | 구현 중단, 계약 수정 |
| G3 로컬 수직기능 | Simulator→판단→출력→DB→UI 통과 | AWS·영상 UI 착수 금지 |
| G4 실제 HIL | 실제 카메라·I/O로 동일 결과 | 어댑터·배선·상태머신 보완 |
| G5 역할·복구 | 3개 모드·백업·복구 통과 | Cloud 범위 확대 금지 |
| G6 AWS | Offline Queue·중복방지·알림 통과 | AWS 기능 최소화 |
| G7 RC | 전체 기능·HIL·성능 승인 | Pilot 설치 금지 |
| G8 Pilot | 현장 인수·30일 모니터링 착수 | 상용 v1.0 보류 |

---

# 4. Phase 0: 범위·저장소·환경 구축

## 4.1 목표

코드 한 줄보다 재현 가능한 개발환경과 변경통제를 먼저 만든다.

## 4.2 산출물

- Private GitHub Monorepo
- `main` Ruleset
- GitHub Project
- Issue Template, PR Template, CODEOWNERS
- Toolchain Version Manifest
- `Makefile`
- CI Skeleton
- Reference IPC·Alternate IPC 후보
- 테스트베드 BOM

## 4.3 완료기준

- 새 PC에서 문서의 순서대로 Clone하고 `make check-prereqs` 가능
- PR 없이 `main` 변경 불가
- D1·T1·T2 계정과 권한 확인
- R3 PR은 T1·T2 승인이 필요하도록 운영규칙 확정
- 비밀정보가 저장소에 없음

---

# 5. Phase 1: AI 어안카메라 API Spike

## 5.1 왜 가장 먼저 하는가

MVP의 가장 큰 외부 위험은 카메라가 광고한 재실기능과 실제 외부 API가 다를 수 있다는 점이다. API가 필요한 데이터를 주지 않으면 Gateway·UI·AWS 설계를 다시 해야 한다.

## 5.2 5일 내 검증항목

### 장치와 스트림

- 고정 IP 설정
- ONVIF Discovery
- RTSP Main/Sub Stream
- H.264 Sub Stream
- NTP 설정
- 카메라 재부팅·복구

### AI 이벤트

- Zone ID
- Person Object Type
- Current Count 또는 Occupied/Empty 상태
- Enter·Exit
- Dwell 또는 체류시간
- Event Start·Update·End
- Confidence 또는 품질정보
- Event Timestamp
- Snapshot 또는 Snapshot URL
- Health·Analytics Status

### 통신

- HTTP Push, Webhook, MQTT, ONVIF Event, 제조사 API 중 실제 사용방식
- 인증방식
- 재연결 후 이벤트 회복
- 중복 이벤트
- 이벤트 순서 역전
- 네트워크 단절 중 카메라 내부 동작

## 5.3 시험 시나리오

1. 빈 구역에서 1명 진입
2. 1명 이탈
3. 2명 동시 진입
4. 구역경계 체류
5. 사람이 설비 뒤에 일부 가려짐
6. 카메라 네트워크 30초 차단
7. 카메라 재부팅
8. 조명 Off·On
9. 빠른 진입·이탈 반복
10. 객체가 화면 가장자리에 위치

## 5.4 Go/No-Go 기준

필수:

- Zone 식별 가능
- 사람 진입 또는 재실을 1초 내 이벤트로 받을 수 있음
- 비재실 또는 이벤트 종료를 판별할 수 있음
- 카메라 장애를 30초 내 판단할 수 있음
- Snapshot 획득 가능
- API 문서 또는 안정적인 실측 프로토콜 확보

No-Go:

- 현재 재실과 누적 입장 수를 구분할 수 없음
- 이벤트 종료가 없어 비재실 전환이 불가능
- 이벤트가 영상 UI에만 보이고 외부 연동 불가
- AI와 스트림을 동시에 사용할 수 없음
- 제조사 기술지원 없이 펌웨어별 동작이 불명확

No-Go이면 카메라를 바꾸고 나머지 개발을 진행하지 않는다.

---

# 6. Phase 2: 계약과 시뮬레이터

## 6.1 계약 우선순위

다음 계약을 먼저 만든다.

```text
contracts/events/camera-event-v1.schema.json
contracts/events/occupancy-state-v1.schema.json
contracts/events/equipment-state-v1.schema.json
contracts/events/safety-decision-v1.schema.json
contracts/events/actuation-result-v1.schema.json
contracts/events/cloud-event-v1.schema.json
contracts/api/local-openapi.yaml
contracts/api/cloud-openapi.yaml
contracts/mqtt/topics.md
```

## 6.2 필수 공통필드

- `schemaVersion`
- `eventId`
- `correlationId`
- `tenantId`
- `siteId`
- `gatewayId`
- `deviceId`
- `zoneId`
- `observedAt`
- `receivedAt`
- `sequenceNo`
- `source`
- `quality`
- `rawReference`

## 6.3 상태 의미

```text
OCCUPIED
VACANT_PENDING
VACANT_CONFIRMED
UNKNOWN
STALE
```

- `OCCUPIED`: 현재 사람이 있다고 판단
- `VACANT_PENDING`: 사람 미검지 전환을 기다리는 단계
- `VACANT_CONFIRMED`: 설정된 연속조건을 통과한 비재실
- `UNKNOWN`: 초기상태, 충돌, 분석불가
- `STALE`: 정해진 시간 동안 신규 데이터 없음

오직 `VACANT_CONFIRMED`만 비재실 조건으로 사용한다.

## 6.4 시뮬레이터

### Camera Simulator

지원 시나리오:

- Occupied·Vacant
- Count 0~5
- Enter·Exit
- Dwell
- Duplicate
- Out-of-order
- Delayed
- Offline
- Malformed Payload

### I/O Simulator

지원 시나리오:

- Equipment Running·Stopped
- Restart Request
- Work Window Input
- Stop Request Output
- Output Feedback
- Timeout
- Modbus Exception
- Network Drop

### Scenario Runner

YAML 또는 JSON 시나리오를 실행한다.

```text
T+0s Equipment=RUNNING
T+1s Zone-A=OCCUPIED
T+1.2s Expected Alarm=ON
T+1.5s Expected StopRequest=PULSE
T+2s Expected Audit=RECORDED
```

## 6.5 완료기준

- 모든 Schema Validation 통과
- Simulator가 정상·오류·장애 이벤트 생성
- Scenario Runner 결과가 CI Artifact로 생성
- T1·T2가 실제 장비 없이 기본 시나리오를 재현

---

# 7. Phase 3: Gateway 로컬 안전 수직기능

## 7.1 패키지 구조

```text
services/gateway-server/
├─ cmd/safegai-edge/
├─ internal/domain/
├─ internal/application/
├─ internal/adapters/camera/
├─ internal/adapters/io/
├─ internal/storage/sqlite/
├─ internal/cloud/outbox/
├─ internal/httpapi/
├─ internal/auth/
└─ internal/observability/
```

## 7.2 구현 순서

### Step 1: Process Skeleton

- Config Load
- Structured Logger
- Graceful Shutdown
- Health Endpoint
- Build Version
- Hardware Profile Check

완료기준:

- systemd 없이 개발 PC에서 실행
- SIGTERM 후 DB 손상 없이 종료
- `/health/live`, `/health/ready` 분리

### Step 2: SQLite

초기 테이블:

- `events`
- `occupancy_states`
- `equipment_states`
- `safety_decisions`
- `actuation_results`
- `audit_logs`
- `cloud_outbox`
- `config_versions`
- `users`

원칙:

- WAL 사용
- 모든 시간 UTC 저장
- Event ID Unique
- Outbox Transactional Insert
- Migration Version 관리

### Step 3: Event Normalizer

- 제조사 Payload를 공통 Event로 변환
- Raw Payload는 제한된 크기로 참조 또는 저장
- 필수필드 누락 시 Reject Queue
- Sequence와 Timestamp 검증
- Duplicate 억제

### Step 4: Occupancy State Machine

기본전환:

```text
UNKNOWN -> OCCUPIED
UNKNOWN -> VACANT_PENDING
OCCUPIED -> VACANT_PENDING
VACANT_PENDING -> VACANT_CONFIRMED
VACANT_PENDING -> OCCUPIED
ANY -> STALE
STALE -> OCCUPIED or VACANT_PENDING
```

금지전환:

- Data Timeout에서 직접 `VACANT_CONFIRMED`
- Camera Offline에서 `VACANT_CONFIRMED`
- Count Parsing Failure에서 `VACANT_CONFIRMED`

### Step 5: Equipment State

- DI 또는 Modbus로 Running·Stopped·RestartRequested 수신
- TTL과 Quality 관리
- 입력이 Stale이면 `UNKNOWN`
- 물리입력과 UI 표시를 동일 Event Store에 기록

### Step 6: Safety Decision

MVP Template 1:

```text
Zone=OCCUPIED AND Equipment=RUNNING
=> WARNING + STOP_REQUEST_REQUIRED
```

MVP Template 2:

```text
RestartRequested AND Zone != VACANT_CONFIRMED
=> RESTART_INTERLOCK
```

MVP Template 3:

```text
Camera/Occupancy=UNKNOWN or STALE
=> SAFETY_CONFIRMATION_UNAVAILABLE
```

MVP Template 4:

```text
ApprovedWorkWindow AND Equipment=STOPPED
=> MAINTENANCE_MONITORING
```

### Step 7: Actuation

- Warning Light Command
- Siren/Audio Command
- Stop Request Pulse
- Command ID와 Correlation ID
- Command Timeout
- Optional Feedback Input
- Retry 제한
- 중복 출력방지

금지:

- 부팅 후 이전 출력명령 자동 재실행
- ACK가 없다는 이유로 무한 Retry
- 일반 DO로 주전원 차단

### Step 8: Local API and WebSocket

- Current Site State
- Camera·Zone·Equipment State
- Active Alarms
- Event List·Detail
- ACK·Resolve·Classify
- Maintenance Work Window
- Maintainer Diagnostics

### Step 9: Cloud Outbox

- Local Transaction과 Outbox 등록
- Exponential Backoff
- Maximum Queue Size Alert
- Idempotency Key
- Dead Letter State
- Cloud 성공 후 상태 업데이트

## 7.3 Gate G3

시뮬레이터로 다음을 자동 통과해야 한다.

- OCCUPIED + RUNNING에서 출력
- VACANT_CONFIRMED + STOPPED에서 출력 없음
- STALE에서 재가동 허용 없음
- Duplicate Event에서 중복 Pulse 없음
- Gateway 재시작 후 과거 Pulse 재실행 없음
- SQLite와 Audit가 일치
- UI에 p95 1초 이내 반영

---

# 8. Phase 4: 실제 카메라·I/O HIL

## 8.1 실제 카메라 어댑터

공통 인터페이스 예:

```text
Connect
Health
SubscribeEvents
GetSnapshot
GetCapabilities
Close
```

제조사별 코드는 `internal/adapters/camera/<vendor>` 안에서만 허용한다.

## 8.2 실제 Modbus I/O

- Coil·Discrete Input 주소표 문서화
- Polling 주기와 Timeout 고정
- Pulse 출력 Duration 검증
- Feedback Input 사용 가능 시 결과확인
- Device Offline을 `UNKNOWN`으로 전환

## 8.3 시험 순서

1. 생산설비와 분리된 Test Relay
2. 램프·부저·릴레이로 출력확인
3. PLC Simulator
4. 실제 PLC의 Test Input
5. 설비 제작사 승인 후 Stop Request 회로

## 8.4 Gate G4

- 실제 카메라 이벤트와 Simulator 결과가 동일한 Domain Event로 변환
- 실제 I/O 지연 p95 500ms 이내
- 네트워크 단절 후 잘못된 출력 없음
- 카메라 재부팅 중 재가동 허용 없음
- T2 HIL Evidence 승인

---

# 9. Phase 5: 사용자·운영자·유지보수자 모드

## 9.1 Frontend 원칙

하나의 React·TypeScript 앱에서 Role에 따라 Navigation과 API 권한을 분리한다. 화면 숨김만으로 권한을 처리하지 않고 Gateway API에서 다시 검증한다.

## 9.2 사용자 모드

필수화면:

- 현재 안전상태
- 1·2·4분할 영상
- 구역별 재실·확인불가
- 설비 운전상태
- 현재 경고
- 행동지침

금지기능:

- 공식 ACK
- 조치종료
- 설정변경
- I/O 시험

## 9.3 운영자 모드

필수화면:

- 위험도순 Event Queue
- Event Image·Zone·Equipment·Action Result
- ACK·Resolve·Classification
- Work Window Start·End
- Device Summary
- 일·주·월 Report

금지기능:

- Safety I/O Mapping
- Safety Rule Meaning 변경
- Remote Machine Control

## 9.4 유지보수자 모드

필수화면:

- Camera Register·Test
- Zone Mapping Preview
- DI Live State
- DO Test in TEST Mode
- Stream Status
- CPU·RAM·Disk·Temperature·NIC
- Backup·Restore
- Update·Rollback
- Diagnostic Bundle

추가보호:

- Local Maintenance Network Only
- Password + Service PIN
- Idle Timeout 15분
- 모든 변경 Audit
- TEST 종료 시 모든 Output Off

## 9.5 Gate G5

- Role별 API 권한시험 통과
- URL 직접입력으로 권한우회 불가
- Kiosk User가 설정에 접근 불가
- TEST 상태 종료 후 Output Off 확인
- T1 UX 승인
- T2 Maintainer HIL 승인

---

# 10. Phase 6: AWS 최소 연계

## 10.1 개발순서

1. CDK Stack Skeleton
2. IoT Thing·Certificate·Policy 개발환경 구성
3. MQTT Status/Event 수신
4. Ingest Lambda
5. DynamoDB Idempotent 저장
6. S3 Event Image
7. SNS Notification
8. Admin API Lambda
9. Cognito
10. Cloud Operator UI

## 10.2 데이터경로

```text
Gateway Local Event
→ SQLite Commit
→ Cloud Outbox
→ MQTT QoS1
→ AWS IoT Rule
→ Ingest Lambda
→ DynamoDB/S3
→ SNS
→ Cloud UI
```

## 10.3 Device Shadow

- `health`: Gateway Report Only
- `settings`: 비안전 Allowlist만 Desired 사용

금지 Desired 항목:

- Stop Request
- Restart Allow
- Safety Rule
- I/O Mapping
- Occupancy State Override

## 10.4 AWS 환경

- `dev`: main merge 후 자동배포
- `pilot`: GitHub Protected Environment 승인 후 배포
- 가능하면 AWS Account 분리
- 최소한 IAM Role·Stack·Data Prefix 분리

## 10.5 Gate G6

- 72시간 Offline Outbox 누적
- 복구 후 중복 없이 재전송
- Event ID Idempotency
- 대표이미지 제한크기 검증
- Gateway 인증서별 Topic 제한
- Cloud에서 Machine Control API 없음
- T1 Cloud Workflow 승인
- T2 WAN Fault 승인

---

# 11. Phase 7: Ubuntu 설치·패키지·복구

## 11.1 Ubuntu Provisioning

- Ubuntu Server 24.04 LTS amd64
- UEFI
- Autoinstall
- Netplan Dual NIC
- Camera Network와 Uplink 분리
- SSH Key Only
- Password Login Off
- Wi-Fi·Bluetooth Default Off
- OS Image ID 기록

## 11.2 Gateway Package

Release Asset:

- `.deb`
- SHA256
- Signature
- SBOM
- Release Manifest
- Rollback Package
- DB Migration Notes
- Supported Hardware Profile
- Qualified Model List

## 11.3 Backup

포함:

- Site·Device·Zone Mapping
- I/O Mapping
- User and Role Metadata
- Event DB 필요범위
- Certificates 제외 또는 별도 보안처리
- Package Version
- OS Image ID

## 11.4 Update·Rollback

1. Package Signature 검증
2. Disk Space 확인
3. DB Backup
4. Service Stop
5. Package Install
6. Migration
7. Health Check
8. 실패 시 이전 Package·DB 복원
9. Audit 기록

## 11.5 Gate

- 전원차단 20회
- Upgrade·Rollback 10회
- Reference→Alternate IPC Restore
- NIC 순서변경 후 역할복구
- 저장공간 90%에서 안전기능 유지
- 부팅 후 과거 출력 재실행 0건

---

# 12. Phase 8: RC와 Pilot

## 12.1 RC 승인조건

- P0/R3 Open Defect 0
- p95 Alarm 1초 이내
- p95 DO 500ms 이내
- 4 Stream 8시간
- 72시간 AWS 단절
- 20회 전원차단
- 100,000 Event DB
- 50,000 Outbox Replay
- 사용자·운영자·유지보수자 승인
- 두 IPC 모델 적합성
- Rollback Evidence

## 12.2 Pilot 준비

- 현장조사서
- Network Plan
- Camera Mount Plan
- Zone Drawing
- I/O Mapping
- PLC/Safety Relay 승인도면
- 개인정보 안내
- Test and Acceptance Plan
- Operator Training
- Maintainer Contact
- Rollback Plan

## 12.3 Pilot 설치순서

1. 생산설비와 분리해 IPC·카메라 설치
2. Camera Network 시험
3. Zone Calibration
4. I/O Input 시험
5. Test Relay Output 시험
6. 운영자 화면 시험
7. AWS 상태·알림 시험
8. 설비 정지 중 PLC Stop Request 시험
9. 승인 후 제한된 운영 시작
10. 30일 모니터링

## 12.4 30일 모니터링 KPI

- Event 수
- 실제위험·정상작업·오감지
- Alarm 실행 성공률
- Stop Request 성공률
- 평균 확인시간
- Camera Offline 시간
- Gateway Reboot 수
- Cloud Queue 최대치
- CPU·RAM·Disk Peak
- 작업자 불편·운영자 개선요청

---

# 13. GitHub Issue 수행절차

모든 기능은 다음 흐름을 따른다.

1. Issue 생성
2. Customer Value 작성
3. Acceptance Criteria 작성
4. Risk R0~R3 지정
5. T1/T2 Test Owner 지정
6. Claude Code `/plan`
7. Contract와 Test 먼저 수정
8. 최소 구현
9. `make verify-fast`
10. Draft PR
11. AI Read-only Review
12. `make verify`
13. T1/T2 증거
14. 승인·Squash Merge
15. Nightly Build

Issue가 2일을 넘으면 더 작은 Issue로 분할한다.

---

# 14. 결함 우선순위

| 등급 | 의미 | 대응 |
|---|---|---|
| P0 | 잘못된 안전출력, 재가동허용, 데이터손상 | 즉시 개발중단, Hotfix |
| P1 | 핵심기능 불가, 카메라·I/O 장시간 장애 | 당일 수정 |
| P2 | 우회 가능한 기능·UX 문제 | 현재 주차 내 수정 |
| P3 | 개선·편의·문서 | Backlog |

P0 예:

- STALE을 VACANT로 표시
- 재부팅 후 과거 Stop Pulse 재실행
- 사용자 모드에서 DO 시험 가능
- Cloud 명령으로 기계제어 가능

---

# 15. 일정 보호규칙

다음 요청이 들어오면 자동으로 2차 Backlog로 이동한다.

- 카메라 5대 이상
- 다중사업장
- 자체 AI 모델
- 실시간 Cloud Video
- 범용 Rule Builder
- 모바일 Native App
- MES·ERP
- 예측분석
- 얼굴인식
- 자동 재가동

Week 1 이후 MVP 범위변경은 제품책임자의 명시적 승인과 일정 재산정이 없으면 허용하지 않는다.
