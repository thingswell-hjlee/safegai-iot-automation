# Architecture Guardrails

이 파일은 SafeGAI 저장소의 아키텍처 경계와 기술 스택을 Kiro가 항상 준수하도록 한다.

## Monorepo Layout

```
services/gateway-server   Go modular-monolith edge runtime (linux/amd64)
services/cloud-backend    TypeScript Lambda handlers and shared cloud domain
apps/frontend             React/TypeScript single frontend with local/cloud adapters
infra/aws                 AWS CDK TypeScript
infra/edge                Edge provisioning and configuration
contracts/                JSON Schema, OpenAPI, event topics
simulators/               Camera and I/O simulators
tests/                    Integration, HIL definitions, acceptance evidence
packaging/gateway         Ubuntu amd64 package and provisioning assets
docs/                     Specifications and runbooks
scripts/                  Build and utility scripts
```

## Technology Stack

| Layer | Technology | Notes |
|-------|-----------|-------|
| Gateway runtime | Go 1.26.x | Exact patch pinned by toolchain and CI |
| Local database | SQLite WAL | No external RDBMS |
| Stream proxy | MediaMTX | No transcoding or recording on gateway |
| Frontend | React + TypeScript | Single app, local/cloud adapters |
| AWS backend | Lambda, DynamoDB, S3, IoT Core, Cognito, HTTP API, CloudFront, SNS | CDK TypeScript |
| Target OS | Ubuntu Server 24.04 LTS x86-64 | Standard UEFI boot |
| Build arch | `linux/amd64` only | Same package on 2 vendor PCs |

## Hard Constraints

- Gateway는 modular monolith를 유지한다. 마이크로서비스 분리는 승인된 ADR 없이 금지한다.
- Gateway에서 영상 AI 추론을 수행하지 않는다.
- 특정 보드 GPIO, 전용 커널, 전용 부트로더, 전용 하드웨어 SDK에 의존하지 않는다.
- 표준 인터페이스만 사용한다: Ethernet, USB, RS485, Modbus, UEFI, SATA/NVMe.
- 외부 의존성이나 AWS 서비스 추가 시 필요성, 보안, 비용, 라이선스, 롤백 영향을 Issue에 설명한다.
- 고객별 소스 분기를 만들지 않는다. 설정과 어댑터로 처리한다.

## Referenced Documents

- #[[file:SOUL.md]]
- #[[file:CLAUDE.md]]
- #[[file:docs/GATEWAY_PRODUCT_SPEC.md]]
- #[[file:docs/PRODUCT_MVP_SPEC.md]]
