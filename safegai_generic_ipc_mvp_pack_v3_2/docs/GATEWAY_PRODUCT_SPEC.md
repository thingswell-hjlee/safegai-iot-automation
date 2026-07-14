# 범용 Ubuntu 산업용 임베디드 PC Gateway 제품 개발사양 v3.0

## 1. 설계 목적
SafeGAI Gateway를 특정 SBC나 제조사에 종속하지 않고, 국내외에서 쉽게 조달 가능한 저가형 소형 x86-64 산업용 임베디드 PC에서 동일하게 동작하도록 한다.

핵심 목표는 다음과 같다.

- 1인 개발자가 표준 Ubuntu와 표준 Linux 도구로 개발·배포한다.
- 특정 보드용 커널, Device Tree, GPIO, 전용 이미지에 의존하지 않는다.
- 기준 IPC가 단종되어도 대체 IPC에 동일 이미지와 동일 패키지를 설치한다.
- 현장 I/O를 PC와 분리하여 Gateway 교체 시 배선과 검증 범위를 줄인다.
- 카메라 4채널 관제, 이벤트 처리, 로컬 UI, AWS 동기화를 저비용 하드웨어에서 안정적으로 수행한다.

## 2. 제품 플랫폼 명칭

- 제품 플랫폼: `SafeGAI Edge IPC-Lite`
- 하드웨어 등급: `SG-EPC-L1`
- 하드웨어 프로파일: `ipc-lite-amd64-v1`
- 운영 아키텍처: `linux/amd64`
- 기준 운영체제: Ubuntu Server 24.04 LTS amd64 Minimal

하드웨어 제조사명이나 특정 모델명은 소스코드와 제품 기능명에 사용하지 않는다.

## 3. 하드웨어 등급

### 3.1 MVP 표준형: IPC-Lite
저가·범용·소형을 우선하는 1차 양산 기준이다.

| 구분 | 필수 사양 |
|---|---|
| CPU Architecture | x86-64/amd64 |
| CPU | 4코어 저전력 프로세서, Intel Processor N100/N97/N150급 또는 동급 이상 |
| RAM | 8GB DDR4/DDR5/LPDDR5 |
| Storage | M.2 SATA 또는 NVMe SSD 128GB 이상 |
| Network | 1GbE 2포트 이상 |
| USB | USB 3.x 2포트 이상 |
| Display | HDMI 1포트 이상 |
| Power | 12V DC 또는 9~36V DC, 자동 전원복구 필수 |
| Cooling | 팬리스 금속 케이스 |
| Install | DIN Rail 브래킷 또는 제어반 고정 가능 |
| Operating Temperature | 0~50°C 범위에서 MVP 시험 통과 |
| BIOS | Power Restore=ON, RTC, USB/SSD Boot |
| Size | 1리터 이하 권장 |
| Average Power | 정상운영 20W 이하 목표 |

### 3.2 산업강화형: IPC-Industrial
고온·전원변동·장기 공급이 중요한 고객을 위한 선택 사양이다.

| 구분 | 권장 사양 |
|---|---|
| Power | 절연 9~36V Wide DC, 잠금형 터미널 |
| Temperature | -20~60°C 이상 |
| Watchdog | Hardware Watchdog |
| Security | TPM 2.0, Secure Boot |
| Serial | 절연 RS232/RS485 |
| Mount | DIN Rail 기본 |
| Storage | 산업용 SSD, Power-Loss Protection 권장 |
| Supply | 3년 이상 공급계획 또는 대체모델 보증 |

MVP 기능은 IPC-Lite에서 모두 동작해야 하며 IPC-Industrial을 필수로 요구하지 않는다.

## 4. 기준 CPU 규칙
CPU는 특정 모델로 고정하지 않고 다음 기능·성능 기준으로 승인한다.

- 64-bit x86
- 물리 또는 효율 코어 합계 4코어 이상
- 최대 동작주파수 3.0GHz급 이상
- AES-NI 또는 동급 암호화 명령 지원 권장
- Ubuntu 기본 커널에서 CPU·GPU·NIC가 동작
- 별도 GPU/NPU 없이 Gateway 전 기능 수행
- Intel N100, N97, N150 또는 AMD 동급은 예시이며 필수 제조사가 아니다.

## 5. 메모리 규칙

- 양산 기준 8GB로 고정한다.
- 4GB 하드웨어는 정식 지원하지 않는다.
- 16GB는 개발·고객 옵션으로 허용하지만 기능 차이를 만들지 않는다.
- 정상운영 시 시스템 전체 메모리 사용량은 4GB 이하, 여유 메모리는 3GB 이상을 목표로 한다.
- Swap은 2GB 이하 또는 zram을 사용하며 지속적인 Swap 발생은 결함으로 판단한다.

## 6. 저장장치 규칙

### 6.1 필수
- M.2 SATA 또는 NVMe SSD 128GB 이상
- SMART 또는 NVMe Health 조회 가능
- microSD와 USB Flash Drive를 운영 저장장치로 사용하지 않음
- OS, DB, 이미지, 로그가 모두 SSD에 저장됨

### 6.2 권장 파티션

| Mount | 권장 크기 | 용도 |
|---|---:|---|
| EFI | 512MB | UEFI Boot |
| `/` | 32GB | Ubuntu와 애플리케이션 |
| `/var/lib/safegai` | 80GB 이상 | DB·이벤트 이미지·Outbox |
| swap/zram | 0~2GB | 비상용 |

### 6.3 저장공간 정책
- 70%: 운영자 Warning
- 80%: 이미지 보관기간 단축 시작
- 85%: 오래된 비증빙 이미지 우선 정리
- 90%: 신규 일반 이미지 저장 제한, 이벤트 메타데이터와 감사로그 유지
- DB 손상·Read-only 전환 시 `DEGRADED` 상태

## 7. 네트워크 규칙

### 7.1 포트 역할
- `LAN1`: Camera/Device LAN
- `LAN2`: Site Uplink, AWS, Maintenance

### 7.2 운영 원칙
- 두 LAN은 서로 다른 서브넷 사용을 기본으로 한다.
- Camera LAN은 인터넷 라우팅을 기본 금지한다.
- DHCP Reservation 또는 고정 IP 지원
- Netplan으로 설정을 버전관리한다.
- Wi-Fi와 Bluetooth는 설치 시에만 선택하고 운영환경에서는 기본 비활성화한다.
- NIC는 Ubuntu 기본 드라이버로 동작해야 하며 외부 DKMS 드라이버 의존을 금지한다.

## 8. 현장 I/O 규칙

### 8.1 구성
- 절연형 Modbus TCP 8DI/8DO를 최우선으로 사용
- Modbus RTU는 절연 USB-RS485 또는 내장 절연 RS485 사용
- Gateway와 I/O 모듈은 명확한 장비 ID와 Register Map을 가진다.

### 8.2 금지
- PC 내장 GPIO로 현장 24V 신호 직접 연결
- USB Relay를 최종 산업용 출력으로 사용
- 일반 Relay로 기계 주전원 직접 차단
- I/O 실패를 정상상태로 간주

### 8.3 안전 출력
```text
SafeGAI Gateway
-> Isolated DO / Modbus Output
-> PLC or Safety Relay Input
-> Approved Machine Safety Circuit
```

## 9. 전원 규칙

### 필수
- 정전 복구 후 BIOS 자동부팅
- 전원 커넥터가 진동으로 쉽게 분리되지 않을 것
- 24V 제어반에서는 24V 입력 IPC 또는 승인된 24V-to-12V DC-DC 사용
- Gateway 전원과 현장 I/O 전원 분리 권장
- 5~10분 보조전원 또는 DC UPS 권장

### 시험
- 저전압·순간정전·강제차단 후 자동복구
- 20회 강제 전원차단
- 부하 중 전원복구
- SSD와 DB 무결성 확인

## 10. 운영체제

### 10.1 기준
- Ubuntu Server 24.04 LTS amd64 Minimal
- UEFI 설치
- Desktop 환경 미설치
- 설치는 `autoinstall`과 cloud-init으로 자동화
- 검증된 Point Release, Kernel, Package Manifest를 `edge-image-id`로 고정

### 10.2 업데이트 정책
- Kernel Major 자동업데이트 금지
- 보안 업데이트는 Dev IPC -> Reference IPC -> Alternate IPC -> Pilot 순서로 검증
- Pilot 환경에서는 유지보수 승인창에서만 적용
- OS Release Upgrade는 MVP 중 금지
- 차기 LTS 전환은 별도 ADR과 30일 회귀시험 후 수행

### 10.3 보안
- SSH Password Login 금지
- SSH Key와 유지보수 LAN만 허용
- UFW 또는 nftables 기본정책 적용
- 기본 인바운드 포트 최소화
- 운영 계정과 서비스 계정 분리
- 카메라·AWS 인증정보는 파일권한과 애플리케이션 암호화로 보호
- TPM 2.0이 있는 모델은 키 보호에 선택 활용

## 11. 프로비저닝 구조

```text
infra/edge/
├─ autoinstall/
│  ├─ user-data
│  └─ meta-data
├─ hardware-profiles/
│  ├─ schema.json
│  ├─ ipc-lite-amd64-v1.yaml
│  └─ qualified-models.yaml
├─ netplan/
├─ systemd/
├─ packaging/
└─ scripts/
   ├─ provision.sh
   ├─ qualify-hardware.sh
   ├─ collect-diagnostics.sh
   └─ rollback.sh
```

## 12. 하드웨어 프로파일 예시

```yaml
profileId: ipc-lite-amd64-v1
architecture: amd64
minimum:
  cpuCores: 4
  memoryMiB: 7600
  storageGiB: 120
  ethernetPorts: 2
  usb3Ports: 2
required:
  uefi: true
  autoPowerOn: true
  hdmi: true
  fanless: true
  stockUbuntuDrivers: true
optional:
  tpm2: true
  hardwareWatchdog: true
  wideDcInput: true
qualification:
  fourStreamHours: 8
  cloudOutageHours: 72
  powerCycles: 20
```

## 13. 실행 프로세스

```text
systemd
├─ safegai-edge.service
│  └─ one Go modular-monolith binary
├─ mediamtx.service
│  └─ RTSP to WebRTC/HLS proxy, pass-through only
├─ safegai-health.service
│  └─ hardware/SSD/network health collector
└─ system services
   ├─ chrony
   ├─ networkd or NetworkManager
   └─ watchdog
```

Docker와 Kubernetes는 Production Gateway에 사용하지 않는다. 개발·CI 시뮬레이터에서는 사용할 수 있다.

## 14. safegai-edge 모듈
- config/version manager
- local auth/RBAC
- camera adapters
- event normalizer
- occupancy state machine
- equipment/DI state
- fixed safety rule evaluator
- alarm and actuator service
- maintenance work-window service
- event/audit store
- outbox/cloud sync
- local REST/WebSocket API
- health/resource monitor
- package update/rollback manager
- hardware profile reader

모듈은 한 프로세스 안에서 명확한 interface로 분리한다.

## 15. 개발 스택
- Go toolchain은 저장소의 `go.mod`와 CI에서 고정
- Linux amd64 native build
- SQLite WAL
- standard `net/http` 또는 경량 router
- JSON Schema 기반 contract validation
- structured JSON log
- systemd watchdog notification
- React/TypeScript 정적파일을 Gateway가 제공

## 16. 영상 사양

### 16.1 카메라 요구
- H.264 보조스트림 필수
- 4분할: 640x360~1280x720, 5~10fps, 채널당 768Kbps 이하 권장
- 단일 확대: H.264 1080p 이하, 2Mbps 이하 권장
- H.265-only 카메라는 MVP 지원대상에서 제외

### 16.2 MediaMTX
- RTSP source pull
- WebRTC 우선, HLS fallback
- codec copy/pass-through
- no recording
- no ffmpeg transcoding
- on-demand connection
- camera credentials는 local encrypted configuration에만 보관

## 17. 저장 사양

### SQLite
- config
- cameras/zones/equipment/I/O mappings
- users/roles
- events/actions/ack/resolve/classification
- occupancy state snapshots
- audit logs
- cloud outbox
- configuration versions
- hardware qualification and health history

### Filesystem
- `/var/lib/safegai/events/YYYY/MM/DD/<event-id>.jpg`
- cloud thumbnail 별도 생성
- 30일 또는 disk quota 기반 정리

## 18. 리소스 예산

| 항목 | 목표 |
|---|---:|
| `safegai-edge` RSS | 400MB 이하 |
| MediaMTX + 4 streams | 1GB 이하 목표 |
| 전체 app memory | 목표 2GB 이하, Release Gate 3GB 이하 |
| 평균 CPU | 목표 35% 이하, Release Gate 40% 이하 |
| CPU p95 | 70% 이하 |
| System reserve | RAM 3GB 이상 |
| Disk write | 정상 평균 1MB/s 이하 목표 |
| Boot to safety-ready | 3분 이내 |
| 4분할 UI | 8시간 연속 무중단 |

## 19. Local API
- `POST /api/v1/session/login`
- `GET /api/v1/system/status`
- `GET /api/v1/system/hardware`
- `GET /api/v1/cameras`
- `GET /api/v1/zones`
- `GET /api/v1/equipment`
- `GET /api/v1/events`
- `POST /api/v1/events/{id}/ack`
- `POST /api/v1/events/{id}/resolve`
- `POST /api/v1/work-windows`
- `POST /api/v1/work-windows/{id}/close`
- `GET /api/v1/maintenance/diagnostics`
- `GET /api/v1/maintenance/hardware-qualification`
- `POST /api/v1/maintenance/io-test`
- `WS /api/v1/realtime`

Maintenance API는 Maintainer Role과 Local Maintenance Network에서만 허용한다.

## 20. 배포·업데이트
- `safegai-edge_<version>_linux_amd64.deb`
- checksum, SBOM, signature, release manifest 포함
- manifest에 `hardwareProfile`, `osImageId`, `minDisk`, `minMemory` 기록
- 현재 버전과 이전 1개 버전 보관
- Update 전에 DB와 Config Snapshot
- Health Check 실패 시 자동 Rollback
- MVP에서는 Cloud가 Update 가능 여부만 표시하고 실제 적용은 Maintainer 승인형

## 21. 하드웨어 교체 절차
1. 새 IPC에 승인된 Ubuntu Image 설치
2. Hardware Qualification Script 실행
3. 기존 Gateway Backup 복원
4. NIC·Camera·I/O Mapping 확인
5. Test Mode에서 I/O 확인
6. T2 승인 후 Normal Mode 전환
7. 이전 IPC는 설정과 로그를 보존하고 폐기·수리 이력 등록

## 22. 양산 Gate
- 기준 IPC와 대체 IPC가 동일 amd64 패키지로 동작
- Ubuntu 기본 드라이버만 사용
- 4채널 스트림 8시간
- 이벤트 10건/초 10분
- DB 100,000건
- Cloud Outbox 50,000건 재전송
- 72시간 WAN 단절
- 20회 전원차단
- SSD Health 조회
- 두 NIC 동시운영
- 평균 CPU 40% 이하, Memory 3GB 이하
- 전원복구 후 3분 이내 Safety Ready
- P0/R3 결함 0건
