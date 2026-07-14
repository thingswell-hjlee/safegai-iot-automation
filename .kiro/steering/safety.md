# Safety Rules

이 파일은 SafeGAI의 절대 안전원칙과 R3 도메인 제약을 정의한다.
Kiro는 이 규칙을 위반하는 코드를 생성하거나 제안하지 않는다.

## Absolute Safety Principles

1. 본 제품은 AI 기반 보조 안전·관제 제품이다. Safety PLC, Safety Relay, 비상정지와 방호장치를 대체하지 않는다.
2. 일반 DO로 기계 동력을 직접 차단하지 않는다.
3. 정지 요청은 기존 PLC, Safety Relay 또는 승인된 안전회로에 전달한다.
4. 카메라 장애·가림·이벤트 지연은 `VACANT`가 아니라 `UNKNOWN` 또는 `STALE`이다.
5. `UNKNOWN`과 `STALE`에서는 재가동 허용을 차단한다.
6. AI 비재실만으로 고위험 설비를 자동 재가동하지 않는다.
7. 비상정지와 기존 안전장치는 SafeGAI보다 항상 우선한다.
8. AWS 장애가 현장 경고·정지 요청에 영향을 주지 않는다.

## Occupancy States

유효한 상태값만 사용한다:

- `OCCUPIED` — 재실 확인됨
- `VACANT_PENDING` — 비재실 대기 (확인 전)
- `VACANT_CONFIRMED` — 비재실 확정
- `UNKNOWN` — 카메라 장애 또는 데이터 없음
- `STALE` — 이벤트 지연 임계 초과

**`VACANT_CONFIRMED`만이 vacancy 조건을 충족한다.**

## R3 Changes

다음 영역의 변경은 Risk Class R3에 해당하며, T1과 T2 모두의 인간 승인 및 HIL 증거가 필요하다:

- 재실 상태머신 전이 로직
- 안전조건 판단 로직
- I/O 매핑 또는 DO 출력 로직
- 정지요청 발생·해제 로직
- 재가동 인터록 조건

Kiro는 R3 영역의 코드를 생성할 수 있지만, 완료로 판단하지 않으며 반드시 인간 검증을 요청한다.

## Forbidden Patterns

- `if state != OCCUPIED { allowRestart() }` — UNKNOWN/STALE에서 재가동 허용 금지
- Camera timeout → state = VACANT — 카메라 장애는 UNKNOWN
- Direct relay power switching for machine shutdown — 일반 DO로 동력 직접 차단 금지
- Cloud-dependent safety execution path — 로컬 안전결정이 AWS에 의존 금지

## Referenced Documents

- #[[file:SOUL.md]]
- #[[file:contracts/safety/]]
