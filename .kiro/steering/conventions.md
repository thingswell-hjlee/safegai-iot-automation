# Coding Conventions

이 파일은 SafeGAI 저장소의 코딩 규약과 파일 구성 원칙을 정의한다.

## Language-Specific Rules

### Go (services/gateway-server)

- Go 1.26.x, CI에서 정확한 패치 버전을 고정한다.
- `gofmt`로 포맷한다.
- `go vet ./...`로 정적분석한다.
- CGO_ENABLED=0으로 빌드한다 (SQLite 제외 시).
- 단일 바이너리 `safegai-edge`로 빌드한다.
- 패키지 구조는 modular monolith를 유지한다.

### TypeScript (services/cloud-backend, apps/frontend, infra/aws)

- 엄격한 TypeScript (strict mode).
- npm을 패키지 매니저로 사용한다.
- `npm run format/lint/typecheck/test --if-present`로 검증한다.

### CDK (infra/aws)

- CDK TypeScript로 모든 AWS 리소스를 정의한다.
- `cdk synth`와 `cdk diff`로 검증한다.
- 리소스 추가 시 비용과 보안을 Issue에 설명한다.

## File Naming

- Go: snake_case 파일명, 패키지명은 소문자
- TypeScript: camelCase 파일명, PascalCase 컴포넌트
- Contracts: kebab-case JSON Schema 파일명
- Docs: UPPER_SNAKE_CASE.md

## Commit Message

```
<type>(<scope>): <short summary>

<optional body>

Refs: #<issue-number>
```

Types: feat, fix, docs, refactor, test, ci, chore

## Error Handling

- Go: 명시적 error 반환, panic 사용 금지 (테스트 제외)
- TypeScript: typed errors, unhandled rejection 금지
- 안전 도메인: 모든 오류는 fail-safe (UNKNOWN/STALE 전이)

## Logging

- 구조화된 로그 (JSON)
- 레벨: DEBUG, INFO, WARN, ERROR
- 안전 이벤트와 상태 전이는 반드시 INFO 이상으로 기록
- 민감 정보(credentials, 개인정보)를 로그에 포함하지 않는다

## Testing

- 안전 도메인은 test-first로 개발한다.
- Contract tests로 인터페이스 호환성을 검증한다.
- 시뮬레이터를 먼저 구현하고, 이후 실제 하드웨어로 교체한다.
- `tests/evidence/`에 시험 증거를 저장한다.

## Referenced Documents

- #[[file:CLAUDE.md]]
- #[[file:docs/TOOLCHAIN_BASELINE.md]]
