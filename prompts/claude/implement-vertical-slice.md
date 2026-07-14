승인된 계획대로 가장 작은 수직기능을 구현하라.

규칙:
- 계약과 실패 테스트를 먼저 작성한다.
- 누락 또는 오래된 카메라 데이터는 VACANT로 처리하지 않는다.
- AWS가 없어도 로컬 안전기능이 동작해야 한다.
- 새 외부 의존성을 최소화하고 추가 이유를 기록한다.
- Vendor 분기는 Adapter 밖으로 유출하지 않는다.
- 구현 후 make verify-fast를 실행한다.
- 완료 전에 read-only reviewer와 test-author 검토를 수행한다.
- push, merge, tag, deploy, 실제 I/O 출력은 수행하지 않는다.
