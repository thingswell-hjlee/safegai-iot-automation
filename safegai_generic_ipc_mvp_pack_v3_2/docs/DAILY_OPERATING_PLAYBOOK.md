# SafeGAI 일일 개발·시험 운영표

## 09:00~09:20 Daily Gate

D1, T1, T2가 다음만 확인한다.

1. 전일 Build Version
2. P0·P1 결함
3. 오늘 구현 Issue 1개
4. 오늘 T1 Acceptance 항목
5. 오늘 T2 HIL 항목
6. Blocking Hardware·Camera·AWS 문제

회의결과를 GitHub Project와 해당 Issue에 기록한다.

## 09:20~10:00 D1 계획

- main 최신화
- Issue Branch 생성
- Claude Code `/plan`
- 변경파일·계약·시험 검토
- Risk에 맞는 승인자 확인

## 10:00~12:00 구현 1

- 실패 Test
- Contract
- 최소 Domain Logic
- Simulator
- `make verify-fast`

## 13:00~15:00 구현 2

- Adapter·Storage·API 또는 UI 연결
- 로그·Audit
- Offline·Restart 동작
- 수동 Smoke

## 15:00 Nightly Candidate

D1:

- Draft PR
- Artifact 또는 설치방법
- T1/T2 시험지시

T1:

- 전일 Build 기능·권한·UX
- 결함등록

T2:

- 전일 Build IPC 설치
- HIL·지연·장애시험
- Evidence 등록

## 16:30 결함분류

- P0: 즉시 현재작업 중지
- P1: 다음 작업 전에 수정
- P2: 해당 주차 내
- P3: Backlog

## 17:00~18:00 마감

D1:

- `make verify`
- PR Update
- 구현·미구현·위험기록
- 다음날 Issue Ready

T1·T2:

- 시험결과와 Build Version 연결
- 재현 가능한 Evidence 확인

## 주간 리듬

### 월요일

- 주간 Gate와 목표 3개 이하
- 신규 기능 동결

### 화~목

- 수직기능 구현·시험
- 하루 1개 PR 원칙

### 금요일

- 오전 기능완성
- 오후 전체 회귀·HIL
- 주간 Demo
- 다음 주 Go/No-Go

## 주간 Demo 내용

- 실제 또는 Simulator Event
- Domain State
- 안전판단
- 출력결과
- 사용자 화면
- Audit
- Failure Scenario 1개

UI만 보여주는 Demo는 완료로 인정하지 않는다.
