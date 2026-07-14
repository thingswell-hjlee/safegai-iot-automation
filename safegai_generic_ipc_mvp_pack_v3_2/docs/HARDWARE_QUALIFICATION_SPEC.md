# SafeGAI 범용 산업용 IPC 하드웨어 적합성 사양 v3.0

## 1. 목적
특정 제조사 모델을 제품의 고정 부품으로 간주하지 않고, 동일한 하드웨어 프로파일을 만족하는 저가형 소형 산업용 PC를 신속하게 승인·대체하기 위한 시험 기준을 정의한다.

## 2. 등급과 프로파일
- 하드웨어 등급: `SG-EPC-L1`
- 소프트웨어 프로파일: `ipc-lite-amd64-v1`

## 3. 승인 단위
하드웨어는 CPU 이름만이 아니라 다음 조합을 하나의 승인 SKU로 관리한다.

- 제조사·모델·Revision
- CPU
- RAM 용량과 구성
- SSD 제조사·모델·용량
- NIC Controller
- BIOS Version
- Power Adapter/DC-DC
- Ubuntu Image ID

구성 변경 시 영향분석 후 부분 또는 전체 재시험한다.

## 4. 제품 운영구성
- `Reference IPC`: 주 양산·시험 기준 1종
- `Alternate IPC`: 단종·납기 대응 대체 1종
- 두 모델 모두 `ipc-lite-amd64-v1`을 만족해야 함
- 초기 출시 중 승인모델은 2종을 넘기지 않음

## 5. 문서심사

| 항목 | 합격 기준 |
|---|---|
| Architecture | x86-64/amd64 |
| CPU | 4코어 이상 |
| RAM | 8GB 이상 |
| SSD | M.2 128GB 이상 |
| NIC | 1GbE 2포트 이상 |
| OS | Ubuntu Server 24.04 LTS 설치 가능 |
| Driver | Ubuntu 기본 Kernel/Repository로 동작 |
| Power Restore | BIOS 자동부팅 지원 |
| Cooling | 팬리스 |
| Temperature | 0~50°C 운영 가능 또는 해당 시험 통과 |
| Mount | 제어반 고정 또는 DIN Rail 가능 |

## 6. 자동 적합성 검사
`qualify-hardware.sh`는 다음을 JSON으로 출력한다.

- CPU Model, Core Count, Architecture
- Total Memory
- Disk Model, Capacity, SMART/NVMe Health
- NIC Count, Link Speed, Driver
- UEFI/BIOS Version
- TPM/Watchdog 존재 여부
- OS Image ID, Kernel Version
- USB Controller
- Thermal Sensor
- Power Restore 시험 결과는 수동 Evidence와 연결

출력 파일:

```text
tests/evidence/hardware/<model>/<date>/qualification.json
```

## 7. 기능 시험
- Ubuntu 자동설치
- 시스템 부팅과 자동로그인 금지
- 두 NIC 고정 IP와 Routing
- Camera LAN 4대 연결
- Modbus TCP/RTU I/O
- HDMI 로컬화면
- USB 진단장치
- RTC/NTP
- systemd 자동시작
- Package Install/Upgrade/Rollback

## 8. 성능 시험

| 시험 | 부하 | 합격 기준 |
|---|---|---|
| 4채널 영상 | H.264 보조스트림 8시간 | 끊김으로 서비스 재시작 없음 |
| 이벤트 폭주 | 10 events/s, 10분 | 유실 0, 중복출력 0 |
| UI 동시접속 | 5 sessions | 주요 API p95 1초 이내 |
| DB | 100,000 events | 조회·기록 정상 |
| Outbox | 50,000 messages | 중복 없이 재전송 |
| CPU | 통합부하 | 목표 평균 35%, Release Gate 평균 40%, p95 70% 이하 |
| Memory | 통합부하 | 목표 App 2GB, Release Gate 3GB 이하, OOM 0 |
| SSD | 24시간 기록 | I/O Error 0 |

## 9. 전원·복구 시험
- 강제 전원차단 20회
- WAN·Camera·I/O 부하 중 전원차단
- 자동부팅 성공 20/20
- Filesystem/SQLite Integrity 정상
- 잘못된 출력 재실행 0건
- 3분 이내 Safety Ready
- BIOS 설정 초기화 여부 확인

## 10. 네트워크 시험
- 각 NIC Cable Disconnect/Reconnect
- NIC Link Flap 50회
- DHCP Failure와 Static Fallback
- Camera LAN Broadcast 증가
- WAN DNS Failure
- AWS 단절 72시간
- 두 NIC Route가 뒤바뀌지 않을 것

## 11. 온도 시험
MVP 기본 시험:
- 실내 5~35°C 실제 운영
- 제어반 모사 40°C에서 8시간 통합부하
- Thermal Throttling, Shutdown, 영상중단 없음
- CPU 온도 Warning Threshold와 측정방법 기록

고온·저온 고객은 IPC-Industrial 별도 프로파일로 시험한다.

## 12. SSD 시험
- SMART/NVMe Health 읽기
- 갑작스러운 전원차단 후 Integrity
- 70/80/85/90% Storage Policy
- Read-only 오류 모의
- SSD 교체와 Backup Restore

## 13. 보안 시험
- Secure Boot 지원 여부 기록
- TPM 2.0 지원 여부 기록
- SSH Password Login 차단
- 불필요 Port 차단
- 초기 Password 변경
- Certificate File Permission
- Factory Reset 시 Credential 제거

## 14. 합격 판정
다음을 모두 만족해야 승인한다.

- 필수 하드웨어 규격 충족
- 모든 P0 기능시험 통과
- 전원 20회, WAN 72시간, 영상 8시간 통과
- Critical Driver/DKMS 의존 없음
- P0/R3 결함 0건
- T2 서명
- 제품책임자 승인

## 15. 변경관리

| 변경 | 재시험 범위 |
|---|---|
| RAM 제조사 동일용량 | Boot·Memory·8h 부하 |
| SSD 모델 | Storage·Power Cycle·24h 기록 |
| NIC Controller | Network 전체 |
| BIOS Version | Boot·Power Restore·NIC·USB |
| CPU 동급 변경 | 전체 성능·온도 |
| Ubuntu Kernel | 전체 회귀·HIL |
| Power Adapter | 전원·부하·전원차단 |
