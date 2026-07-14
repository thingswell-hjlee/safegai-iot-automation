# START_HERE.md
## SafeGAI 개발 착수 순서

이 문서는 개발자 D1, 기능시험자 T1, 성능·HIL 시험자 T2가 처음 저장소를 받은 날부터 따라야 하는 실행 순서이다.

---

## 1. 첫 번째 의사결정

개발을 시작하기 전에 다음을 고정한다.

| 항목 | 확정값 |
|---|---|
| 제품 | SafeGAI AI Fisheye Zone Safety 4CH |
| 적용범위 | 하나의 위험공정 |
| AI 카메라 | 어안 1대 필수, 전체 최대 4대 |
| Gateway | `SG-EPC-L1`, `ipc-lite-amd64-v1` |
| OS | Ubuntu Server 24.04 LTS amd64 |
| Gateway | Go Modular Monolith |
| Local DB | SQLite WAL |
| Frontend | React + TypeScript 단일 앱 |
| AWS | IoT Core, Lambda 2개, DynamoDB 2개, S3, Cognito, HTTP API, CloudFront, SNS |
| 현장 I/O | 외장 절연형 Modbus 8DI·8DO |
| 개발조직 | D1 1명, T1 1명, T2 1명 |
| 일정 | 8주 RC + 2주 Pilot |

위 값이 변경되면 코딩부터 하지 말고 `SOUL.md`, 제품사양, ADR과 GitHub Issue를 먼저 변경한다.

---

## 2. 개발 순서

반드시 다음 순서로 개발한다.

1. GitHub 저장소와 개발환경 구축
2. AI 어안카메라 API 5일 검증
3. 공통 이벤트·재실·안전결정 계약 확정
4. 카메라·I/O 시뮬레이터 구축
5. 로컬 안전 수직기능 완성
6. 실제 카메라·실제 Modbus I/O 교체
7. 사용자·운영자·유지보수자 모드
8. AWS 최소 연계
9. Ubuntu 설치·패키지·백업·롤백
10. 테스트베드 인수와 현장 Pilot

다음 단계로 넘어갈 때마다 Gate를 통과한다. Gate를 통과하지 못한 상태에서 다음 기능을 병렬 추가하지 않는다.

---

## 3. 첫 번째 수직기능

첫 구현 목표는 다음 한 흐름이다.

```text
카메라 시뮬레이터가 Zone A OCCUPIED 이벤트 발생
AND I/O 시뮬레이터가 Equipment RUNNING 상태 제공
→ Gateway가 이벤트 정규화
→ Occupancy State = OCCUPIED
→ Safety Decision = STOP_REQUEST_REQUIRED
→ Test Output 실행
→ SQLite Event/Audit 저장
→ 사용자 화면에 1초 이내 표시
→ Cloud Outbox 등록
```

이 흐름이 시뮬레이터에서 통과한 후 실제 카메라와 실제 I/O를 연결한다.

---

## 4. 오늘 해야 할 일

### D1

- Private GitHub Monorepo 생성
- 이 패키지 반영
- `main` 보호규칙과 GitHub Project 생성
- Toolchain 확인
- `make check-prereqs` 실행
- Issue 001~005 생성
- Reference IPC Ubuntu 설치 준비

### T1

- 사용자·운영자 모드의 핵심 업무 10개 작성
- 각 업무의 예상 결과와 금지 동작 작성
- `tests/acceptance/`에 초안 저장

### T2

- 테스트베드 BOM 확정
- Reference IPC·대체 IPC 후보 확인
- 어안카메라·Modbus I/O·시험용 램프·부저·릴레이 확보
- 전원차단·네트워크차단 시험방법 확정

---

## 5. 절대 먼저 만들지 않을 것

- 4채널 완성형 대시보드
- 클라우드 실시간 영상
- 범용 Rule Editor
- 다중현장 SaaS
- 예측 AI
- 자체 영상 추론
- MES·ERP 연동
- 자동 재가동

이 기능은 MVP 성공과 직접 관계가 없고 1인 개발의 일정 위험을 크게 만든다.

---

## 6. 다음 문서

- Day 0~5: `docs/DAY_0_TO_DAY_5_RUNBOOK.md`
- 전체 개발: `docs/DEVELOPMENT_EXECUTION_GUIDE.md`
- 초기 Issue: `docs/GITHUB_INITIAL_BACKLOG.md`
- 테스트베드: `docs/TESTBED_SETUP_RUNBOOK.md`
- Claude Code: `docs/CLAUDE_CODE_AUTO_PLAYBOOK.md`
