# SafeGAI 1차 제품 최종 개발사양 v3.0

## 1. 제품명
**SafeGAI AI Fisheye Zone Safety 4CH**

## 2. 제품목적
AI 어안카메라 중심으로 하나의 위험공정 내 사람의 재실·진입·체류를 확인하고 설비 운전·재가동 상태와 결합하여 현장 경고, 안전정지 요청, 관리자 확인, 클라우드 증빙을 제공한다.

Gateway는 특정 SBC가 아니라 범용 저가형 소형 Ubuntu 산업용 임베디드 PC를 사용한다.

## 3. 제품 안전등급 경계
- SafeGAI는 AI 기반 보조 안전·관제 제품이며 인증된 Safety PLC, Safety Relay, 비상정지, 방호장치를 대체하지 않는다.
- 출력은 승인된 안전회로에 정지 또는 재가동 차단을 요청하며 기계 주전원을 직접 개폐하지 않는다.
- 카메라 비재실 판단만으로 설비 자동 재가동을 허용하지 않는다.

## 4. 정량 사양

| 항목 | 확정값 |
|---|---:|
| AI 어안카메라 | 1대 필수 |
| 전체 카메라 | 최대 4대 |
| 위험/주의/작업 구역 | 최대 8개 |
| 설비 | 최대 4대 논리상태 |
| DI/DO | 각 8점 |
| Gateway CPU | x86-64 4코어 저전력급 |
| Gateway RAM | 8GB |
| Gateway SSD | M.2 128GB 이상 |
| Gateway LAN | 1GbE 2포트 |
| 사용자 동시접속 | 로컬 5명 |
| 클라우드 관리자 | 현장당 5명 |
| 로컬 이벤트 보관 | 30일 |
| 클라우드 이벤트 보관 | 기본 365일 |
| 대표 이미지 | 이벤트당 최대 1장, Cloud 96KB 이하 |
| Cloud outage buffer | 7일 이상 |
| 독립운전 검증 | 72시간 이상 |
| 연속운전 | 30일 |

## 5. 핵심 수직기능

```text
Camera occupancy/event
-> normalize
-> occupancy state
-> machine DI state
-> fixed safety rule
-> lamp/siren/stop-request
-> local event and audit
-> local UI
-> asynchronous AWS event/image/status
```

## 6. 고정 안전 규칙

### R-01 운전 중 위험구역 재실
- 조건: `zone=OCCUPIED` AND `machine=RUNNING`
- 동작: 적색 경광등, 음성경고, 이벤트, 설정에 따른 정지요청

### R-02 재가동 요청 중 재실
- 조건: `restart_request=ON` AND zone != `VACANT_CONFIRMED`
- 동작: PLC/Safety Relay에 재가동 차단 요청, 경고, 운영자 확인요청

### R-03 카메라 안전확인 불가
- 조건: `UNKNOWN` OR `STALE`
- 동작: 안전확인 불가 표시, 재가동 허용 금지, 장비장애 알림

### R-04 정비 작업창
- 조건: 승인된 작업창 활성
- 동작: 작업창 상태 표시, 진입이력 유지, 자동정지 정책은 현장 승인설정 적용, 종료 시 잔류자 확인

### R-05 중복 이벤트
- 동일 Camera/Zone/Type/Time-window 이벤트는 한 건으로 통합한다.
- 한 이벤트로 출력명령이 중복 실행되지 않아야 한다.

## 7. 재실 상태머신
- `OCCUPIED`: 사람 1명 이상 확인
- `VACANT_PENDING`: 비재실 샘플 수집 중
- `VACANT_CONFIRMED`: 설정시간 또는 연속샘플로 비재실 확인
- `UNKNOWN`: 카메라·API·구역 상태 판정 불가
- `STALE`: 최신 데이터가 허용시간 초과

기본값:
- Vacancy Confirm: 3초 또는 3회 연속
- Occupancy Stale: 10초
- Camera Offline: 30초
- 중복 억제: 2초

현장시험 후 파라미터를 확정하되 의미 자체는 변경하지 않는다.

## 8. Gateway 플랫폼
- Hardware Grade: `SG-EPC-L1`
- Hardware Profile: `ipc-lite-amd64-v1`
- OS: Ubuntu Server 24.04 LTS amd64 Minimal
- Packaging: Signed `.deb`
- Service: systemd
- App: Go Modular Monolith
- DB: SQLite WAL
- Stream Proxy: MediaMTX Pass-through
- Frontend: React Static App
- 현장 I/O: 외장 절연형 Modbus TCP/RTU

## 9. 로컬 관제
- 1/2/4분할 H.264 보조스트림
- 이벤트 채널 강조와 단일 확대
- 현재 안전상태, 구역 재실, 설비 상태
- 활성 경보, 미확인 이벤트
- 네트워크·카메라·I/O·AWS 상태
- 사용자·운영자·유지보수자별 화면 분리

## 10. 클라우드 최소연계
- Gateway Last-seen, 버전, Hardware Profile, CPU/RAM/SSD
- 이벤트 메타데이터
- 대표 이미지
- 운영자 이메일/SMS 선택 알림
- 이벤트 확인상태 동기화
- 원격 읽기전용 조회와 보고
- 안전제어, I/O 시험, 재가동 명령은 제공하지 않는다.

## 11. 보고서
- 일/주/월 이벤트 건수
- 구역별·카메라별·설비별 건수
- 실제 위험/정상작업/오감지 분류
- 경고·정지요청 성공/실패
- 평균 확인시간
- 반복 위험시간대
- 장비 장애와 미조치 이벤트
- Gateway Hardware/SSD Health 요약

## 12. 완료 기준
- 카메라 4대 동시 연결
- 어안카메라 8구역 상태수신
- 경고 p95 1초
- DO p95 500ms
- 8GB amd64 Gateway Resource Budget 통과
- 기준 IPC와 대체 IPC 동일 패키지 동작
- 72시간 Cloud Outage
- 20회 Power Cycle
- T1 기능 인수와 T2 HIL·Hardware Qualification 인수
