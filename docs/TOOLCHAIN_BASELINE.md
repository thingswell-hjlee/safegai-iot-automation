# SafeGAI Toolchain Baseline
## 기준일: 2026-07-14

도구 버전은 MVP 중 자동으로 변경하지 않는다. 보안 또는 치명적 결함이 아니라면 RC 이후 업그레이드한다.

| 영역 | 기준 |
|---|---|
| Gateway Go | 1.26.5 |
| Node.js | 24 LTS |
| Lambda Runtime | nodejs24.x |
| AWS CDK | v2, package-lock으로 정확히 고정 |
| Ubuntu Gateway | Ubuntu Server 24.04 LTS amd64 |
| GitHub Checkout Action | v7 major, 운영 시 Commit SHA Pin 권장 |
| GitHub setup-go | v6 major |
| GitHub setup-node | v7 major |
| Claude Code | 팀이 검증한 동일 버전, 자동 업데이트 후 Smoke Test |
| Kiro | 팀이 검증한 동일 버전 |

## 버전 고정 파일

- `services/gateway-server/go.mod`
- `services/gateway-server/go.sum`
- Root `package.json`
- `package-lock.json`
- `.github/workflows/*.yml`
- Release Manifest
- Ubuntu OS Image ID

## 업그레이드 절차

1. 별도 Issue 생성
2. 변경 이유와 지원기간 기록
3. Dependency·Security Scan
4. Unit·Integration Test
5. Reference IPC Smoke
6. Alternate IPC Smoke
7. AWS Dev Deploy
8. Rollback 확인
9. T1 또는 T2 승인
10. Merge

## 금지

- `latest` Container Tag
- Version Range가 넓은 핵심 Runtime Dependency
- Pilot 직전 Major Upgrade
- GitHub Action을 검토 없이 자동 Major Update
