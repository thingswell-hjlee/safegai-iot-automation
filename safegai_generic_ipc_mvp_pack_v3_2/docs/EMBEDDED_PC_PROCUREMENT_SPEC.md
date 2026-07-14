# SafeGAI 범용 소형 Ubuntu 산업용 임베디드 PC 조달사양 v3.0

## 1. 조달 목적
특정 브랜드나 모델을 지정하지 않고 최소 2개 공급사에서 대체 조달 가능한 SafeGAI Gateway IPC를 선정한다.

## 2. 등급과 프로파일
- 하드웨어 등급: `SG-EPC-L1`
- 소프트웨어 프로파일: `ipc-lite-amd64-v1`
- 실제 제조사·모델은 Qualified Gateway List에서 관리한다.

## 3. 필수 조달사양

| 항목 | 필수 기준 | 확인방법 |
|---|---|---|
| Hardware Profile | `ipc-lite-amd64-v1` | Profile 검사 |
| CPU | x86-64 4코어, N100/N97/N150급 또는 동급 이상 | `lscpu`, 부하시험 |
| RAM | 8GB 이상 | `free`, `dmidecode` |
| SSD | 교체형 M.2 SATA/NVMe 128GB 이상 | 분해자료, `smartctl`/`nvme` |
| LAN | RJ45 1GbE 2포트 이상, Ubuntu 기본 Driver | `ethtool`, `lspci`, `iperf3` |
| USB | USB 3.x 2포트 이상 | 실제 연결시험 |
| Display | HDMI 1포트 이상 | 실제 출력시험 |
| 전원 | 12V 또는 9~36V DC, 자동 전원복구 | 강제 전원시험 |
| UEFI | Ubuntu 표준 ISO 설치 가능 | 설치시험 |
| 냉각 | Fanless 금속케이스 | 8시간 부하시험 |
| 설치 | 벽면/VESA/DIN 브래킷 가능 | 기구검토 |
| OS | Ubuntu Server 24.04 LTS amd64 정상동작 | 적합성 시험 |
| 온도 | 0~50°C 범위 시험 통과 | Testbed |

## 4. 우선 구매사양
- 256GB SSD
- 9~36V Wide Input
- TPM 2.0
- Hardware Watchdog
- 내장 절연 RS485
- DIN Rail Mount
- -20~60°C 동작
- KC·EMC 등 판매지역 관련자료
- 3년 이상 공급계획
- BIOS 설정 Backup 또는 동일 BIOS 유지정책

## 5. 공급사 제출자료
- 제품 Datasheet
- CPU·RAM·SSD·LAN 상세사양
- BIOS Manual
- 전원 Adapter 인증자료
- 판매지역 적합성 자료
- Ubuntu/Linux 호환 근거
- 제품 공급기간과 단종 통보기간
- A/S·RMA 기간
- SSD 교체방법
- Operating Temperature

## 6. 샘플 구매전략
- 공급사 또는 독립모델 A: 2대
- 공급사 또는 독립모델 B: 2대
- DUT, Reference, Recovery 용도로 사용
- 동일 Software Package와 Test Case로 비교
- Pilot은 1개 승인모델로 가능하지만 상용 v1.0 전 2개 모델 승인을 원칙으로 한다.

## 6. 비용관리 원칙
- RAM 8GB와 SSD 128GB를 기본가격에 포함
- Windows License 없는 Barebone 또는 Ubuntu 사전설치형 우선
- PoE Switch와 절연 I/O는 PC와 분리 조달
- 16GB RAM, 256GB SSD, Wide Input은 Option SKU로 분리
- PC 본체가격보다 교체성·드라이버·납기·A/S를 총비용으로 평가

## 8. 입고검사
- 외관·Serial Number
- BIOS Version
- AC Power Recovery
- RAM·SSD 용량
- LAN 2포트
- USB 3.x
- Ubuntu Install
- 30분 CPU·Memory Stress
- 4채널 Stream 2시간
- Reboot 5회
- SMART/NVMe Health
- Device Certificate Provisioning 준비

## 9. 합격판정
필수항목이 모두 통과하고 동일 Package로 기능·성능시험을 통과한 모델만 Qualified Model List에 등록한다.
