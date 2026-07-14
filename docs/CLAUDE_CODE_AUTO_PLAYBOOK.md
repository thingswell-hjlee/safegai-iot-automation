# Claude Code Auto 운영 Playbook
## SafeGAI 1인 개발 가속화 기준

Claude Code는 구현속도를 높이는 도구이며 제품범위·안전판단·릴리스 승인 주체가 아니다.

---

# 1. 설정 구조

## 프로젝트 공유설정

```text
.claude/settings.json
```

포함:

- 허용 명령
- 승인요청 명령
- 차단 명령
- Hook

프로젝트 저장소는 Auto Mode를 스스로 활성화하는 수단으로 사용하지 않는다.

## 사용자설정

```text
~/.claude/settings.json
```

예:

```json
{
  "$schema": "https://json.schemastore.org/claude-code-settings.json",
  "permissions": {
    "defaultMode": "auto"
  }
}
```

저장소의 `.claude/settings.user.example.json`을 참고하되 기존 사용자 설정과 병합한다.

---

# 2. 사용 금지

- `bypassPermissions`
- `--dangerously-skip-permissions`
- Root 또는 Sudo 상태에서 Claude 실행
- Production AWS Account 직접작업
- 인증서·Private Key가 있는 폴더 접근
- 실제 설비에 연결된 상태에서 자동 I/O 시험

---

# 3. Issue 시작 절차

## Step 1: Issue 확인

```bash
gh issue view <NUMBER>
```

확인:

- Customer Value
- Acceptance Criteria
- Risk R0~R3
- Test Owner
- Rollback

## Step 2: Branch

```bash
git switch main
git pull --ff-only
git switch -c feature/<NUMBER>-<slug>
```

## Step 3: Claude Plan

Claude Code 실행 후:

```text
/plan
```

기본 Prompt:

```text
Issue #<NUMBER>를 수행한다.
먼저 SOUL.md, CLAUDE.md, AIDLC.md, 관련 사양과 가장 가까운 component CLAUDE.md를 읽어라.
아직 파일을 수정하지 말고 다음을 제시하라.
1. 요구사항과 금지사항
2. 변경 파일
3. 계약 변경 여부
4. 실패모드와 오프라인 동작
5. 자동시험
6. T1/T2 수동시험
7. 롤백
범위를 Issue Acceptance Criteria 밖으로 확장하지 마라.
```

## Step 4: Plan 승인

다음 조건이면 승인하지 않는다.

- Microservice 추가
- 새 Database 추가
- Vendor-specific 로직이 Domain에 침투
- AWS 의존 Safety Path
- 범용 Rule Editor
- 테스트 없는 구현
- R3 의미 변경

승인 후 Auto로 구현한다.

---

# 4. 구현 Prompt 패턴

## Gateway 기능

```text
승인된 계획대로 가장 작은 수직기능만 구현하라.
계약과 실패 테스트를 먼저 작성하고, 구현 후 make verify-fast를 실행하라.
카메라 데이터가 없거나 오래되면 VACANT로 처리하지 마라.
새 외부 의존성을 추가하면 이유와 대안을 PR 메모에 기록하라.
```

## 안전상태·I/O

```text
이 변경은 R3이다. 구현 전 상태전이표와 Truth Table을 작성하라.
과거 명령 재실행, 중복 Pulse, Timeout, 재부팅, I/O Offline 테스트를 포함하라.
실제 장비 명령은 실행하지 말고 Simulator Test만 수행하라.
완료 후 safety-reviewer를 읽기전용으로 실행하라.
```

## AWS

```text
AWS Dev Stack의 최소 변경만 구현하라.
cdk synth와 cdk diff까지만 자동 실행하고 deploy/destroy는 실행하지 마라.
IoT Policy는 Gateway 인증서별 Topic으로 제한하라.
Cloud-to-device Machine Control API 또는 Topic을 만들지 마라.
```

## Frontend

```text
Role별로 화면과 API 권한을 모두 검증하라.
사용자 모드는 ACK·설정·I/O 시험을 할 수 없어야 한다.
UNKNOWN과 STALE을 정상 또는 비재실 색상으로 표시하지 마라.
키보드와 큰 글자 환경을 포함한 테스트를 작성하라.
```

---

# 5. Subagent 사용

권장:

- `gateway-builder`: Go 구현
- `cloud-builder`: Lambda·CDK
- `frontend-builder`: React
- `test-author`: Failure Test
- `safety-reviewer`: 읽기전용 R2/R3 검토
- `release-auditor`: Release Evidence

한 Issue에서 구현 Subagent를 동시에 2개 이상 사용하지 않는다. Contract 파일은 Main Session만 수정한다.

---

# 6. Hook

## SessionStart

- 핵심문서 목록
- Current Branch
- Git Status
- 변경금지 경로

## PostToolUse

- 변경파일 포맷
- 생성파일 최소검사

## Stop

- `make verify-fast`
- 실패하면 완료로 보고하지 않음

Hook은 외부 URL, Secret, AWS Deploy를 호출하지 않는다.

---

# 7. 구현 종료 절차

Claude에게 요청:

```text
구현을 종료하기 전에 다음을 수행하라.
1. 변경사항을 Acceptance Criteria에 매핑
2. make verify 실행
3. 실패모드와 오프라인 동작 설명
4. 보안·권한·하드웨어 영향 설명
5. 롤백 절차 작성
6. T1과 T2가 수행할 수동시험 작성
7. PR 본문 초안 작성
아직 push, merge, tag, deploy하지 마라.
```

D1 검토:

- `git diff`
- 신규 Dependency
- Error Handling
- Logging
- Secret
- Test Quality
- Resource Impact

---

# 8. PR 생성

```bash
make verify
git status
git add <reviewed-files>
git commit -m "feat: <short description>"
gh pr create --draft --fill
```

Auto가 Push·Merge·Tag·Deploy하지 않도록 한다.

PR 필수기록:

- Issue
- Risk
- AI 사용영역
- Human Review
- Test Commands
- T1/T2 Manual Test
- Hardware Impact
- Rollback

---

# 9. 안전한 자동화 경계

| 작업 | Auto | 사람 승인 |
|---|---:|---:|
| Source·Test 수정 | O |  |
| Format·Lint·Unit Test | O |  |
| Simulator 실행 | O |  |
| CDK Synth·Diff | O |  |
| Local Build·Package Dry-run | O |  |
| Product Scope 문서 |  | O |
| Occupancy 의미 |  | O |
| Safety I/O |  | O |
| AWS Deploy·Destroy |  | O |
| Actual HIL Output |  | O |
| Git Push·Merge·Tag |  | O |
| Pilot Release |  | O |

---

# 10. 문제가 생길 때

Claude가 반복 수정하면:

1. 세션 중지
2. `git diff` 확인
3. Issue 범위 재확인
4. 실패 Test를 사람이 명확하게 작성
5. 새 Session에서 `/plan`

대규모 변경을 한 번에 되돌리기 위해 `git reset --hard`를 Auto에 허용하지 않는다. 검토된 Commit 또는 `git restore <file>`을 사람이 선택해 사용한다.
