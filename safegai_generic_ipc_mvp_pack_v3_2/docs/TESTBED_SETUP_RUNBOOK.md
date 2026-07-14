# SafeGAI 테스트베드 구축 Runbook
## T2 성능·HIL 시험 기준

테스트베드는 실제 생산설비에 연결하기 전에 카메라, Gateway, I/O, 출력, 전원, 네트워크 장애를 안전하게 재현하는 장치이다.

---

# 1. 테스트베드 목적

- 실제 위험 없이 안전조건과 출력 검증
- Reference·Alternate IPC 비교
- 카메라 API와 재실 정확도 검증
- Modbus I/O 지연과 장애 검증
- 전원·네트워크·저장장치 장애 재현
- GitHub HIL Workflow와 시험증거 자동화

---

# 2. 권장 BOM

| 구분 | 수량 | 요구사항 |
|---|---:|---|
| Reference IPC | 1 | SG-EPC-L1 기준 |
| Alternate IPC | 1 | 다른 제조사, 동일 Profile |
| AI 어안카메라 | 1 | 기준 모델·Firmware 고정 |
| 보조 AI 카메라 | 1 | 4채널 확장 Smoke Test |
| PoE Switch | 1 | 관리형 권장 |
| Modbus TCP/RTU 8DI·8DO | 1 | 절연형 |
| 24V DC 전원 | 1 | I/O·램프용 |
| IPC 전원 | 2 | 제조사 권장 정격 |
| 황색 표시등 | 1 | DO 시험 |
| 적색 표시등 | 1 | DO 시험 |
| 부저·사이렌 | 1 | DO 시험 |
| Test Relay | 2 | Stop Request·Feedback |
| Toggle Switch | 4 | DI 시뮬레이션 |
| 비상정지형 스위치 | 1 | 기존 안전장치 우선성 모사 |
| 전원 반복 장치 | 1 | 원격 PDU 또는 타이머 |
| QA Runner PC | 1 | GitHub Self-hosted Runner |
| 온도계 또는 센서 | 1 | IPC 표면·함체 온도 |

---

# 3. 물리 배선

```text
[PoE Switch]
 ├─ AI Fisheye Camera
 ├─ Auxiliary Camera
 ├─ Reference IPC camera0
 └─ QA PC Test Network

[Office/Uplink Network]
 ├─ Reference IPC uplink0
 └─ QA PC

[24V PSU]
 ├─ Modbus I/O
 ├─ Yellow Lamp
 ├─ Red Lamp
 ├─ Buzzer
 └─ Test Relay

[DI]
 DI1 Equipment Running Switch
 DI2 Restart Request Switch
 DI3 Stop Request Feedback
 DI4 Work Window Physical Confirmation

[DO]
 DO1 Yellow Warning
 DO2 Red Warning
 DO3 Buzzer
 DO4 Stop Request Test Relay
```

생산설비의 동력회로에 연결하지 않는다.

---

# 4. 안전수칙

- DO는 Test Relay와 저전압 표시장치에만 연결
- 220V 부하는 인증된 중계릴레이 없이 직접 연결 금지
- I/O 전원과 IPC 전원을 분리
- 비상정지 스위치는 Gateway와 독립적으로 Test Output 전원을 끌 수 있도록 구성
- 실제 PLC 연결은 출력논리와 전압을 설비 담당자가 승인한 이후 수행
- TEST 상태에서만 수동 DO 시험
- TEST 종료 시 모든 DO Off 확인

---

# 5. 네트워크 장애 주입

시험방법:

1. Camera Ethernet 제거
2. PoE Port Disable
3. `tc netem`으로 지연·손실 주입
4. Uplink 제거
5. DNS 차단
6. AWS Endpoint 차단

측정:

- Camera Offline 판정시간
- Occupancy `STALE` 전환시간
- Local Alarm 지속여부
- Outbox 증가량
- 복구 후 Replay 시간
- 중복 Event 수

---

# 6. 전원 장애 시험

## 강제 전원차단 20회

각 회차:

1. Gateway 정상운전 확인
2. 임의 Event 발생
3. 전원차단
4. 10초 후 전원복구
5. BIOS Auto Power On
6. Safety Ready 시간 측정
7. SQLite Integrity 확인
8. 과거 DO Pulse 재실행 여부 확인

합격:

- 20회 모두 자동부팅
- 3분 내 Safety Ready
- DB 손상 0
- 과거 Pulse 재실행 0

---

# 7. 지연 측정

시간점:

```text
T0 Camera Event observedAt
T1 Gateway receivedAt
T2 Safety Decision createdAt
T3 I/O command sentAt
T4 Output feedbackAt
T5 UI displayedAt
```

계산:

- Camera→Gateway = T1-T0
- Gateway Decision = T2-T1
- Decision→DO = T3-T2
- End-to-End Output = T4-T0
- Local UI = T5-T0

최소 100회 반복하고 p50·p95·max를 기록한다.

---

# 8. 재실 정확도 기본시험

시나리오:

- 1명 진입·이탈 100회
- 2명 동시 진입 30회
- 경계 체류 30회
- 설비 가림 30회
- 조명변화 30회
- 빠른 통과 30회

기록:

- True Occupied
- False Occupied
- Missed Occupied
- Vacant Confirmation Delay
- Zone별 결과
- Camera Firmware와 설정

MVP에서는 사람 신원추적 정확도를 요구하지 않는다. 안전조건에 필요한 재실·비재실·확인불가 상태를 검증한다.

---

# 9. 4채널 영상 시험

조건:

- 4개 H.264 Substream
- 5~10fps
- 채널당 권장 768Kbps 이하
- WebRTC 우선
- 8시간 연속

측정:

- Gateway CPU 평균·p95
- 메모리 시작·종료
- IPC 온도
- Stream 재연결 횟수
- Browser 메모리
- 화면 Freeze

합격:

- 평균 CPU 40% 이하
- p95 70% 이하
- App Memory 3GB 이하
- 치명적 Freeze 0

---

# 10. 시험증거 구조

```text
tests/evidence/<issue-or-release>/
├─ environment.md
├─ wiring.png
├─ versions.json
├─ test-plan.md
├─ measurements.csv
├─ screenshots/
├─ logs/
├─ result.md
└─ approval.md
```

`environment.md` 필수내용:

- IPC Model·BIOS
- CPU·RAM·SSD
- Ubuntu Image ID·Kernel
- SafeGAI Version
- Camera Model·Firmware
- I/O Model·Firmware
- Network Topology
- Test Date·Tester

---

# 11. HIL GitHub Runner

원칙:

- 전용 QA PC 사용
- 외부 Fork PR 실행 금지
- Production AWS Credential 없음
- `workflow_dispatch` 수동실행
- Runner Label 예: `self-hosted,linux,x64,safegai-hil`
- 시험 후 결과만 Artifact로 업로드
- Runner에서 `bypassPermissions` 사용 금지

---

# 12. 실제 설비 연결 전 Gate

모두 충족해야 한다.

- Test Relay 100회 출력 성공
- 중복 Pulse 0
- Camera Offline에서 재가동 허용 0
- I/O Offline에서 안전상태 오판 0
- TEST 종료 Output Off
- 전원차단 20회
- T1 사용자·운영자 기능 승인
- T2 HIL 승인
- 설비 담당자의 전기·PLC 도면 승인
