# Development Workflow

이 파일은 SafeGAI 저장소에서 Kiro가 따라야 하는 개발 워크플로우와 품질 Gate를 정의한다.

## Work Sequence

1. Issue와 인수기준을 읽는다.
2. 관련 코드를 탐색하고 짧은 구현 계획을 작성한다.
3. 인터페이스가 변경되면 `contracts/` 계약을 먼저 갱신한다.
4. 도메인과 안전 동작에 대한 실패 테스트를 먼저 작성한다.
5. 가장 작은 수직 기능 단위로 구현한다.
6. 반복 중 `make verify-fast`를 실행한다.
7. PR 전에 `make verify`를 실행한다.
8. R2/R3 변경은 Safety Reviewer에게 검토를 요청한다.
9. docs와 changelog를 업데이트한다.
10. PR 증거를 준비한다. 직접 병합하지 않는다.

## Branch Strategy

- Trunk-based development with short feature branches.
- 하나의 Issue는 하나의 Branch와 하나의 PR로 처리한다.
- Branch naming: `feat/<issue-number>-<short-description>` 또는 `fix/<issue-number>-<short-description>`

## Verification Commands

| Command | 용도 |
|---------|------|
| `make check-prereqs` | 개발도구 설치 확인 |
| `make format` | 코드 포맷팅 |
| `make lint` | Linter 실행 |
| `make typecheck` | Type checking |
| `make test` | Unit tests |
| `make test-contract` | Contract validation |
| `make test-integration` | Integration tests |
| `make security` | Secret/credential scan |
| `make build` | Build all components |
| `make verify-fast` | format + lint + typecheck + test + contract + security |
| `make verify` | verify-fast + build + integration |

**명령을 임의로 만들지 않는다.** Makefile에 정의된 명령만 사용한다.

## Risk Classification

| Class | 범위 | 승인 |
|-------|------|------|
| R0 | 문서, 포맷, 비제품 도구 | T1 또는 제품책임자 |
| R1 | UI, 보고서, 읽기전용 API | T1 + CI |
| R2 | 카메라 어댑터, 동기화, 인증, 장비관리, 하드웨어 추상화 | T1 또는 T2 + 통합시험 |
| R3 | 재실 상태, 안전조건, I/O, 정지요청, 재가동 인터록 | T1 + T2 + HIL 증거 |

## Forbidden Actions

Kiro는 다음을 수행하지 않는다:

- Pull Request를 병합하거나 Release Tag를 생성
- Pilot·Production에 배포
- 비밀번호, API Key, 인증서, Private Key를 출력하거나 Commit
- IAM, 인증서, 네트워크 정책, 프로덕션 데이터를 변경
- 파괴적 데이터베이스 마이그레이션 실행
- 제품 범위, 게이트웨이 하드웨어 계약, 안전 정책 문서를 인간 승인 없이 수정

## Referenced Documents

- #[[file:CLAUDE.md]]
- #[[file:AIDLC.md]]
- #[[file:Makefile]]
