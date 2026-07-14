# Day 0~Day 5 개발 착수 Runbook

이 문서는 첫 6영업일 동안 그대로 실행할 작업표이다. 첫 주의 목표는 코드를 많이 만드는 것이 아니라 **카메라 API와 개발체계의 불확실성을 제거하는 것**이다.

---

# Day 0: 착수회의와 범위 동결

## 오전 1: 제품 목적 확인

참석: D1, T1, T2, 제품책임자

확정할 문장:

> 사람이 위험구역에 있거나 안전확인이 불가능한 상태에서 설비가 운전되거나 재가동되는 위험을 줄인다.

확정할 MVP:

- 하나의 위험공정
- AI 어안카메라 1대
- 전체 카메라 최대 4대
- 최대 8개 Zone
- 8DI·8DO
- Warning·Stop Request
- 사용자·운영자·유지보수자 모드
- AWS 상태·이벤트·이미지·알림·백업

## 오전 2: 안전경계 서명

회의록에 다음을 기록한다.

- Safety PLC·Safety Relay·비상정지를 대체하지 않음
- AI 비재실 단독 자동 재가동 금지
- Cloud Machine Control 금지
- Camera Offline은 `STALE`
- 일반 DO로 기계 주전원 차단 금지

## 오후: GitHub 준비

### 저장소 이름

권장:

```text
safegai-platform
```

### 로컬 생성 예

```bash
mkdir safegai-platform
cd safegai-platform
git init -b main
```

GitHub CLI 사용 예:

```bash
gh repo create <ORG>/safegai-platform --private --source=. --remote=origin
```

기존 패키지를 저장소 Root에 복사한다.

```bash
rsync -a <PACK_PATH>/ ./
```

초기 Commit:

```bash
git add .
git commit -m "chore: establish SafeGAI MVP product and delivery baseline"
git push -u origin main
```

### GitHub 설정

- Private Repository
- Issues On
- Projects On
- Actions On
- Secret Scanning On
- Dependabot Alerts On
- Discussions Off for MVP
- Wiki Off; 문서는 저장소에서 관리

### main Ruleset

- Require Pull Request
- Require conversation resolution
- Require status checks
- Block force push
- Block deletion
- Squash merge only

### Day 0 산출물

- 착수회의록
- 범위동결 결정
- GitHub 저장소
- Initial Commit
- T1·T2 Collaborator 등록

### Day 0 완료조건

- 세 사람이 저장소를 Clone 가능
- `SOUL.md`와 안전경계를 모두 확인
- 신규기능 요청은 Backlog로 이동하는 규칙 합의

---

# Day 1: 개발 PC와 도구 준비

## D1 개발환경 기준

권장 Host:

- Ubuntu 24.04 LTS 또는 Windows 11 + WSL2 Ubuntu 24.04
- VS Code
- Claude Code
- Kiro IDE
- GitHub CLI
- AWS CLI v2
- Go 1.26.x
- Node.js 24.x
- npm
- Make
- jq
- sqlite3
- shellcheck
- Docker Engine/Compose: 개발 시뮬레이터용 선택

Production Gateway에는 Docker를 사용하지 않는다.

## 버전 확인

```bash
git --version
gh --version
go version
node --version
npm --version
aws --version
make --version
jq --version
sqlite3 --version
shellcheck --version
claude --version
```

실행:

```bash
make check-prereqs
```

버전결과를 다음 파일에 기록한다.

```text
docs/evidence/dev-environment-<DATE>.md
```

## Claude Code 사용자 설정

프로젝트 저장소가 스스로 Auto Mode를 활성화하도록 의존하지 않는다. 사용자 설정에 적용한다.

```bash
mkdir -p ~/.claude
cp .claude/settings.user.example.json ~/.claude/settings.json
```

이미 사용자 설정이 있으면 덮어쓰지 말고 `permissions.defaultMode`만 병합한다.

프로젝트 `.claude/settings.json`은 Allow·Ask·Deny·Hook 정책을 공유한다.

## VS Code Workspace Trust

- 저장소를 신뢰하기 전에 `.claude/settings.json`과 Hook Script 검토
- Hook이 외부 Network 또는 Secret에 접근하지 않는지 확인
- `.env`, 인증서, Private Key가 Git Ignore인지 확인

## Kiro 준비

- Workspace Root를 저장소로 Open
- `.kiro/steering/` 확인
- `.kiro/specs/aws-mvp/` 확인
- AWS Spec은 Requirements부터 검토
- Kiro와 Claude Code가 같은 Branch·파일을 동시에 수정하지 않음

## Day 1 GitHub Issue

```text
#001 Repository and Toolchain Bootstrap
```

Acceptance:

- D1 개발 PC에서 `make check-prereqs` 성공
- Claude Project Policy Load 확인
- Kiro Spec 파일 확인
- 비밀정보 Scan 0건

## Day 1 완료조건

- D1이 새 Branch를 만들고 Commit 가능
- T1·T2가 Build Artifact를 내려받을 수 있음
- Toolchain 버전이 문서화됨

---

# Day 2: GitHub Workflow와 CI Skeleton

## 오전: Project Board

Column:

- Backlog
- Ready
- In Development
- Functional Test
- HIL / Hardware Qualification
- Ready to Release
- Done

Field:

- Risk R0~R3
- Test Owner
- Component
- Hardware Dependency
- Target Week
- Release

## Issue 생성

최소 생성:

1. Repository Bootstrap
2. Camera API Spike
3. Common Event Contract
4. Camera Simulator
5. I/O Simulator
6. Gateway Skeleton
7. Occupancy State Machine
8. Safety Rule Vertical Slice
9. Local User Mode
10. AWS Minimal Stack

상세내용은 `docs/GITHUB_INITIAL_BACKLOG.md` 사용.

## CI Skeleton

PR마다 최소 확인:

- Markdown·JSON·YAML Syntax
- Secret Scan
- Shellcheck
- Hardware Profile Schema
- Go Test: 모듈이 생긴 이후
- TypeScript Lint/Test: 앱이 생긴 이후
- CDK Synth: 인프라가 생긴 이후

초기에는 디렉터리가 없다는 이유로 CI가 실패하지 않도록 Guard를 사용한다.

## Branch 작업 예

```bash
gh issue view 1
git switch -c feature/1-repository-bootstrap
make verify-fast
git add .
git commit -m "chore: add repository bootstrap and CI skeleton"
gh pr create --draft --title "chore: bootstrap SafeGAI repository" --body-file .github/PULL_REQUEST_TEMPLATE.md
```

PR Template를 실제 내용으로 수정한다.

## T1 작업

다음 사용자업무를 Acceptance Case로 작성:

- 현재 안전상태 확인
- 활성 경고 확인
- 작업자 행동지침 확인
- 운영자 Event ACK
- 운영자 조치종료
- 운영자 오감지 분류
- 유지보수 기능 접근차단

## T2 작업

테스트베드 배선도 초안:

- IPC Dual LAN
- Camera PoE Switch
- Modbus I/O
- 24V Power
- Test Lamp
- Buzzer
- Test Relay
- DI Toggle Switch
- Power Cycle Device

## Day 2 완료조건

- 첫 PR이 CI를 통과
- GitHub Project에 Issue가 표시
- T1 Acceptance Case 초안
- T2 Testbed Drawing 초안

---

# Day 3: IPC와 테스트베드 준비

## Reference IPC 설치

확인:

- x86-64 4 Core
- RAM 8GB
- SSD 128GB+
- Dual 1GbE
- Auto Power On
- Ubuntu Stock Driver
- Fanless

## BIOS

설정:

- Restore on AC Power Loss = Power On
- UEFI Boot
- Secure Boot: 사용여부를 제품정책에 따라 고정
- Wake on LAN: 필요 시
- USB Boot: 유지보수 정책에 따라 제한

BIOS Version과 설정을 기록한다.

## Ubuntu 설치

초기 개발은 수동 설치로 시작하되 Autoinstall과 동일한 파티션·사용자·Network 원칙을 적용한다.

필수:

- Hostname: `safegai-gw-dev01`
- UTC Time
- NTP
- SSH Key
- Password SSH Off
- `camera0`, `uplink0` NIC 역할
- Wi-Fi Off
- Automatic Major Upgrade Off

## Network

예:

```text
camera0: 192.168.50.10/24
uplink0: DHCP 또는 현장관리망
```

Camera Network에서 Internet Routing을 기본 차단한다.

## Modbus I/O 시험

생산설비 연결 금지.

- DI 1: Equipment Running Switch
- DI 2: Restart Request Switch
- DI 3: Output Feedback
- DO 1: Yellow Lamp
- DO 2: Red Lamp
- DO 3: Buzzer
- DO 4: Test Stop Relay

## T2 기본시험

- 8DI Read
- 8DO On·Off
- Pulse 500ms·1s
- Modbus Timeout
- I/O Power Off
- IPC Reboot 후 Output Default Off

## Day 3 완료조건

- IPC Hardware Profile 1차 통과
- Ubuntu 설치·NIC 역할 확인
- Test Relay 출력 확인
- 실제 설비와 물리적 분리 확인

---

# Day 4: AI 어안카메라 API 검증 1

## 카메라 초기설정

- Firmware Version 기록
- 관리자 Password 설정
- NTP
- Fixed IP
- H.264 Sub Stream
- Zone 2개 설정
- AI Person Detection On
- Snapshot On

## API 조사 순서

1. 제조사 공식 문서
2. ONVIF Capability
3. Event Subscription
4. HTTP·Webhook·MQTT·SDK
5. RTSP Main/Sub Stream
6. Snapshot API
7. Health API

## Raw Capture

모든 이벤트 원본을 저장한다.

```text
tests/evidence/camera-spike/<model>/<firmware>/
├─ capability.json
├─ occupied-event.json
├─ vacant-event.json
├─ enter-event.json
├─ exit-event.json
├─ health-online.json
├─ health-offline.md
└─ screenshots/
```

개인정보가 포함된 실제 작업자 영상 대신 시험자를 사용하고 외부배포하지 않는다.

## 테스트

- Zone A 진입·이탈 20회
- Zone B 진입·이탈 20회
- 2명 진입
- 빠른 통과
- 10초 체류
- Camera Reboot
- Network 30초 차단

## 측정

- Event Observed Time
- Gateway/Client Received Time
- Snapshot Availability
- Duplicate Rate
- Missing Rate
- Recovery Time

## Day 4 완료조건

- 최소 6종 Raw Event 확보
- Zone ID·Person·Timestamp 확인
- 장애와 복구 이벤트 확인

---

# Day 5: 카메라 API Gate와 계약 초안

## 오전: 분석

다음 표를 작성한다.

| 항목 | 지원 | 실제 필드 | 지연 | 한계 |
|---|---|---|---:|---|
| Zone Occupied |  |  |  |  |
| Current Count |  |  |  |  |
| Enter |  |  |  |  |
| Exit |  |  |  |  |
| Dwell |  |  |  |  |
| Snapshot |  |  |  |  |
| Health |  |  |  |  |
| Event End |  |  |  |  |

## Gate Meeting

참석: D1, T1, T2, 제품책임자

결정:

- GO: 기준 카메라로 확정
- CONDITIONAL GO: 부족 필드를 Gateway 상태머신으로 보완
- NO-GO: 다른 카메라로 교체

Conditional Go 허용 예:

- 현재 Count는 없으나 Occupied Start·End가 안정적
- Dwell은 Gateway에서 계산 가능

No-Go 예:

- Event End 없음
- Zone ID 없음
- External API 없음
- 장시간 이벤트 누락

## 계약 초안

카메라 원본을 다음 공통형식으로 변환한다.

```json
{
  "schemaVersion": "1.0",
  "eventId": "...",
  "cameraId": "cam-01",
  "zoneId": "zone-a",
  "eventType": "OCCUPANCY_CHANGED",
  "occupied": true,
  "personCount": 1,
  "observedAt": "...",
  "receivedAt": "...",
  "sequenceNo": 101,
  "quality": "VALID",
  "snapshotRef": "...",
  "source": "vendor-adapter"
}
```

## Day 5 GitHub 산출물

- Camera Spike Report
- ADR: 기준 카메라 선정
- Common Camera Event Draft
- Capability Matrix
- Go/No-Go Decision

## Day 5 완료조건

- G1 Camera Gate 통과
- Week 2 구현범위 확정
- 카메라 불확실성이 Backlog에 명시
