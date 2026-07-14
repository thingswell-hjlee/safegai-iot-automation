# 사용자·운영자·유지보수자 모드 사양 v3.0

## 1. 구분 원칙
UI 역할과 현장 시스템 상태를 분리한다.

### UI 역할
- `USER`: 사용자 모드
- `OPERATOR`: 운영자 모드
- `MAINTAINER`: 유지보수자 모드

### 현장 시스템 상태
- `NORMAL`: 정상 운영
- `WORK_WINDOW`: 승인된 정비·청소 작업창
- `TEST`: 유지보수 I/O 시험
- `DEGRADED`: 카메라·I/O·SSD·NIC 일부 장애

## 2. 권한표

| 기능 | 사용자 | 운영자 | 유지보수자 |
|---|:---:|:---:|:---:|
| 현재 안전상태 | O | O | O |
| 1/2/4 영상 | O | O | O |
| 활성 경보 확인 | O | O | O |
| 공식 이벤트 확인(Ack) | X | O | O |
| 조치·종료·분류 | X | O | O |
| 보고서 | 제한 | O | O |
| 작업창 요청/종료 | X | O | O |
| 카메라 등록 | X | X | O |
| 구역 매핑 | X | X | O |
| DI/DO 매핑 | X | X | O + 승인 |
| 출력 개별시험 | X | X | O + TEST 상태 |
| 로그/진단 | X | 요약 | O |
| IPC Hardware/SSD/NIC 상세 | X | 요약 | O |
| 백업/복원 | X | X | O |
| 업데이트/롤백 | X | X | O + 승인 |
| Hardware Profile 변경 | X | X | 제품 릴리스로만 |
| 안전규칙 의미 변경 | X | X | 제품 릴리스로만 |

## 3. 사용자 모드
### 목적
작업자가 복잡한 설정 없이 현재 위험과 조치방법을 즉시 이해한다.

### 화면
- 전체 안전상태
- 4분할 영상 또는 선택 영상
- 구역별 정상/재실/확인불가
- 설비 운전상태
- 현재 경고와 행동지침
- 긴급 연락정보

### UX
- 큰 글자와 버튼
- 색상 + 아이콘 + 텍스트
- 기술용어 최소화
- `UNKNOWN/STALE`을 안전으로 표시하지 않음
- 읽기전용 Kiosk Session 선택 가능
- 공식 Ack·조치 권한 없음

## 4. 운영자 모드
### 목적
위험 이벤트를 확인하고 조치·종료·보고한다.

### 화면
- 위험도순 이벤트 큐
- 이미지, Zone, Equipment State, Executed Actions
- Ack/Resolve/Classification
- 작업창 생성·종료
- 카메라·I/O·AWS·Gateway 요약상태
- SSD 저장공간과 Gateway Online 상태
- 일/주/월 보고서

### 제한
- Safety I/O Mapping 변경 불가
- 정지요청 로직 의미 변경 불가
- Hardware Profile 변경 불가
- 원격 설비제어 불가

## 5. 유지보수자 모드
### 목적
설치, 시험, 장애진단, 하드웨어 교체, 업데이트를 수행한다.

### 접근
- Local Maintenance Network Only in MVP
- Username/Password + Service PIN
- Idle Timeout 15 Minutes
- 모든 변경 감사로그
- 안전 I/O Mapping 변경은 Signed Config Version과 T2/제품책임자 승인기록 필요

### 화면
- Camera Discovery/Register/Test
- Zone Mapping and Event Preview
- DI Live State and Mapping
- DO Test in TEST State
- MediaMTX Stream Status
- CPU/RAM/SSD/Temperature/NIC
- Hardware Profile, Model, BIOS, OS Image ID
- SMART/NVMe Health
- Logs and Diagnostics Bundle
- Config Version Backup/Restore
- Update/Rollback
- Hardware Replacement Wizard

### Hardware Replacement Wizard
1. 새 IPC Hardware Qualification 확인
2. Backup File 검증
3. NIC 역할 매핑
4. Camera·I/O 연결시험
5. TEST 상태 출력시험
6. T2 승인기록
7. NORMAL 전환

### TEST 상태 안전규칙
- TEST 진입 전에 운영자 확인
- 생산설비와 물리적으로 분리하거나 Approved Test Relay 사용
- 실제 Stop Request Output은 별도 2단계 확인
- TEST 종료 시 모든 Output Off와 상태 재동기화

## 6. Local Authentication
- Password Hash: Argon2id
- Local Session Token
- Role-based API Authorization
- Login Failure Rate Limit
- Initial Password Change Required
- Cloud Cognito Account와 Local Account는 MVP에서 분리
