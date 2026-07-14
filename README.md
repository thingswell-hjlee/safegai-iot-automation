# SafeGAI AI Fisheye Zone Safety

AI 어안카메라와 범용 x86-64 Ubuntu 산업용 임베디드 PC를 이용하여 하나의 위험공정에서 재실·설비 상태를 결합하고, 현장 경고·안전정지 요청·관리자 알림·안전 증빙을 제공하는 1차 산업안전 MVP 저장소이다.

## 먼저 읽을 문서

1. `START_HERE.md`
2. `SOUL.md`
3. `AIDLC.md`
4. `docs/PRODUCT_MVP_SPEC.md`
5. `docs/DEVELOPMENT_EXECUTION_GUIDE.md`
6. `docs/DAY_0_TO_DAY_5_RUNBOOK.md`

## MVP 목표

- Week 8: 테스트베드 통과 Release Candidate
- Week 10: 첫 현장 Pilot Release
- Pilot 30일: v1.0 출시 판정

## 절대 원칙

- Gateway에서 영상 AI 추론을 개발하지 않는다.
- AWS 장애가 현장 경고와 정지 요청에 영향을 주지 않는다.
- AI 비재실만으로 설비를 자동 재가동하지 않는다.
- 일반 DO로 기계 주전원을 직접 차단하지 않는다.
- 카메라 데이터가 없으면 `VACANT`가 아니라 `UNKNOWN` 또는 `STALE`이다.

## 저장소 상태

이 패키지는 개발 착수용 문서·규칙·자동화 스캐폴드이다. 제품 기능 구현은 GitHub Issue와 Pull Request를 통해 단계적으로 추가한다.
