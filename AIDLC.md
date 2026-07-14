# AIDLC.md
## SafeGAI 1개발자·2시험자 AI-Driven Development Lifecycle v3.0

## 1. 목적
Claude Code Auto, 전문 Subagent, GitHub Actions, 카메라·I/O 시뮬레이터와 실제 범용 x86-64 산업용 PC 테스트베드를 결합하여 8주 MVP RC와 10주 Pilot Release를 달성한다.

## 2. 사람 역할
### D1 개발자/PL
- 제품·소프트웨어 아키텍처
- Gateway, AWS, Frontend 개발
- 하드웨어 추상화, 계약, DB, 배포, 패키징
- 결함 수정과 릴리스 후보 생성
- AI 에이전트 작업의 범위 설정과 최종 코드 책임

### T1 기능·UX 시험자
- 사용자·운영자 모드 인수기준
- 요구사항 기반 기능시험
- UI 접근성·오조작·권한시험
- 회귀시험과 결함 재현
- 운영 매뉴얼·현장 인수서 검증

### T2 성능·HIL·테스트베드 시험자
- SG-EPC-L1 산업용 PC, 카메라, Modbus DI/DO 테스트베드
- 지연, CPU, 메모리, SSD, Dual LAN, 네트워크 성능
- 카메라 단절·전원반복·클라우드 단절·이벤트 폭주
- Safety Relay/PLC 정지요청과 피드백
- 하드웨어 호환성·교체·복구·현장 장시간 운전시험

## 3. AI 역할
### Kiro IDE
- AWS 기능의 requirements/design/tasks 명세
- `.kiro/steering`을 통한 고정 아키텍처·안전경계 적용
- CDK 설계 검토; 배포와 안전범위 변경은 수행하지 않음

### VS Code + Main Claude Code Auto Session
- GitHub Issue 단위의 계획·통합·구현
- `auto` 모드로 반복 구현과 검증
- 인간 승인 Gate에서 정지

### Gateway Builder Agent
- Go domain, hardware profile, adapters, SQLite, local API, sync

### Cloud Builder Agent
- Lambda, DynamoDB, IoT contracts, CDK-compatible code

### Frontend Builder Agent
- role-based React UI and local/cloud API adapters

### Test Author Agent
- unit, contract, integration, hardware-abstraction, failure and acceptance tests

### Safety Reviewer Agent
- read-only adversarial review of R2/R3 changes

### ChatGPT
- 제품범위·ADR·위험분석·시험행렬·PR/릴리스 독립검토
- 결정사항은 GitHub Issue/Spec/PR로 전환된 후에만 유효

AI 리뷰는 인간 T1/T2 승인과 동일하지 않다.

## 4. 단일 진실원
- 제품 원칙: `SOUL.md`
- AI 운영규칙: `CLAUDE.md`
- Lifecycle: `AIDLC.md`
- 최종 제품사양: `docs/PRODUCT_MVP_SPEC.md`
- Gateway 사양: `docs/GATEWAY_PRODUCT_SPEC.md`
- Hardware 승인: `docs/HARDWARE_QUALIFICATION_SPEC.md`
- AWS 사양: `docs/AWS_MVP_SPEC.md`
- 역할·모드: `docs/ROLE_MODE_SPEC.md`
- 요구사항/설계/작업: `docs/specs/<feature>/`
- 계약: `contracts/`
- 하드웨어 프로파일: `config/hardware-profile.example.yaml`
- 업무추적: GitHub Issues/Projects
- 검증증거: GitHub Actions artifacts와 `tests/evidence/`

## 5. Risk Class
### R0
문서, 포맷, 비제품 도구. T1 또는 제품책임자 승인.

### R1
UI, 보고서, 읽기전용 API. T1 승인과 CI.

### R2
카메라 어댑터, 동기화, 인증, 장비관리, 업데이트, 하드웨어 추상화. T1 또는 T2 승인과 통합시험.

### R3
재실 상태, 안전조건, I/O 매핑, 정지요청, 재가동 인터록. T1·T2 모두 승인, HIL 증거, 제품책임자 출시 승인.

## 6. 개발 Gate
### Gate 0 범위·하드웨어 등급 동결
- 공정 1개
- 카메라 1~4대
- 구역 최대 8개
- I/O 맵
- SG-EPC-L1 최소사양
- 성공 KPI와 제외기능

### Gate 1 Specification
모든 Issue는 다음을 포함한다.
- 고객가치
- 사용자·운영자·유지보수자 영향
- 기능·비기능 요구사항
- 실패·오프라인 동작
- 하드웨어 의존성과 지원범위
- 인수기준
- Risk class

### Gate 2 Design
- 데이터 흐름과 상태머신
- API/Schema
- 하드웨어 프로파일과 Capability
- 리소스 예산
- 보안·권한
- 장애·복구
- 시험전략
- 롤백

### Gate 3 AI-assisted Implementation
- 짧은 feature branch 또는 worktree
- 계약 우선
- 안전 도메인 test-first
- Claude Auto로 구현·포맷·반복시험
- 보호파일·배포·안전변경·하드웨어 최소규격 변경은 인간승인

### Gate 4 Automated Verification
- format/lint/typecheck
- unit/contract/integration
- secret/dependency/SAST
- `linux/amd64` build와 `.deb` package
- hardware profile validation
- frontend build
- CDK synth/diff
- resource budget checks where applicable

### Gate 5 T1 Functional Acceptance
- 사용자·운영자 workflow
- 권한·오조작
- 이벤트 확인·조치·보고
- UI 회귀

### Gate 6 T2 HIL/Performance/Compatibility Acceptance
- 카메라 이벤트·폴링·단절
- DI/DO·피드백·timeout
- 4채널 영상
- CPU/RAM/SSD/Dual LAN
- 클라우드 단절·재연결
- 전원반복
- SG-EPC-L1 승인모델 2종 동일 패키지 설치

### Gate 7 Pilot Release
- 서명된 `linux/amd64` 패키지
- 배포 manifest·SBOM·checksum
- 이전 버전 rollback 패키지
- 설치·인수 체크리스트
- Hardware Compatibility Report
- 72시간 offline test
- 20회 power cycle
- 현장 30일 모니터링 시작

## 7. PR 승인규칙
- 개발자는 자신의 R2/R3 PR을 최종 승인하지 않는다.
- R1: T1 1명
- R2: T1 또는 T2 1명 + 관련 시험증거
- R3: T1과 T2 모두 + HIL report
- Pilot AWS 배포: T2 또는 제품책임자 환경승인
- SG-EPC-L1 최소사양 변경: T2 + 제품책임자 승인

## 8. 일일 운영
- 오전: D1이 Ready Issue 1개 선택, 인수기준 고정
- D1: Claude Code Auto + Subagents로 구현
- T1: 전일 Nightly Build 기능시험과 결함 Issue
- T2: 전일 amd64 Package를 테스트베드에서 시험
- 오후: 자동검증, PR, 시험증거 연결
- 매일 Main 통합, 매주 금요일 RC Tag

## 9. 하드웨어 후보 승인 흐름
1. 구매 전 Datasheet와 Ubuntu Driver 위험 검토
2. 1대 입고 후 BIOS·NIC·SSD·Serial·Watchdog Capability Report 생성
3. 승인된 Ubuntu Image ID로 Autoinstall과 Provisioning 수행
4. 4채널·I/O·전원·WAN 장애시험
5. T2가 Candidate Report 등록
6. 동일 소프트웨어 패키지와 Hardware Profile만으로 통과하면 Approved List에 등록
7. 소스수정이 필요하면 후보를 제외하거나 별도 ADR 승인

## 10. 출시 Exit Criteria
- 모든 P0/R3 시나리오 통과
- 현장경고 p95 1초 이내
- DO 요청 p95 500ms 이내
- SG-EPC-L1 8GB 최저 승인모델 정상운전
- 승인모델 2종에 동일 패키지 설치
- 72시간 Cloud Outage
- 20회 Power Cycle
- 이벤트 중복으로 인한 중복 출력 0건
- 재부팅 후 Unsafe Action Replay 0건
- Critical/High 보안결함 0건
- T1/T2 서명된 인수증거
