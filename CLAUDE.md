# CLAUDE.md
## SafeGAI MVP Repository Instructions v3.0

Read these files before planning or changing code:
1. `START_HERE.md`
2. `SOUL.md`
3. `AIDLC.md`
4. `docs/PRODUCT_MVP_SPEC.md`
5. `docs/GATEWAY_PRODUCT_SPEC.md`
6. `docs/DEVELOPMENT_EXECUTION_GUIDE.md`
7. The nearest component-level `CLAUDE.md`

## Team Context
This repository is developed by one human developer and validated by two independent human testers:
- T1: functional/UX/regression acceptance
- T2: performance/HIL/testbed/field acceptance

AI agents may implement, test, review, and document work, but they are not human approvers.

## Delivery Goal
Produce an 8-week MVP release candidate and a 10-week pilot release for one industrial process using:
- one AI fisheye camera plus up to three auxiliary cameras
- a vendor-neutral low-cost x86-64 Ubuntu industrial embedded PC, 8GB baseline
- isolated DI/DO and PLC/Safety Relay stop-request integration
- local user/operator/maintainer modes
- minimal AWS status, event, image, notification, and backup services

## Gateway Hardware Contract
- Production architecture is `linux/amd64`.
- Minimum CPU class is a low-power x86-64 4-core processor, Intel Processor N100/N97/N150 class or equivalent.
- Minimum memory is 8GB; 16GB is optional and must not be required by MVP features.
- Storage is replaceable M.2 SATA or NVMe SSD, 128GB minimum.
- Two 1GbE RJ45 ports are required.
- Only standard UEFI, Ethernet, USB, RS485, Modbus, SATA/NVMe and Linux interfaces may be assumed.
- Do not introduce vendor-specific GPIO, custom kernel, bootloader, board SDK, or proprietary hardware daemon dependencies.
- The same package must pass on two qualified PC models from different vendors.

## Architecture Decisions
- Private monorepo and trunk-based development
- One Go 1.26.x modular-monolith gateway binary; exact patch pinned by toolchain and CI
- SQLite WAL and filesystem event media
- MediaMTX only as a stream proxy; no gateway transcoding or recording
- One React/TypeScript application with local and cloud adapters
- AWS serverless: IoT Core, two Lambda handlers, two DynamoDB tables, S3, Cognito, HTTP API, CloudFront, SNS
- CDK TypeScript for all AWS resources
- Ubuntu Server 24.04 LTS x86-64 baseline with automated provisioning, not a board-specific image

## Required Occupancy States
- `OCCUPIED`
- `VACANT_PENDING`
- `VACANT_CONFIRMED`
- `UNKNOWN`
- `STALE`

Only `VACANT_CONFIRMED` can satisfy a vacancy condition. `UNKNOWN` and `STALE` fail safe.

## Non-negotiable Rules
1. Never implement video AI inference on the gateway.
2. Never make local alarm or stop-request execution depend on AWS.
3. Never directly switch machine power with a general-purpose relay.
4. Never treat missing camera data as vacancy.
5. Never permit automatic restart from AI vacancy alone.
6. Never expose direct cloud machine-control APIs in MVP.
7. Never store credentials, certificates, `.env` files, production IPs, or personal data in Git.
8. Keep the gateway a modular monolith; do not create microservices without an approved ADR.
9. Do not add a generic rule editor in MVP. Implement approved rule templates only.
10. Do not add hardware-vendor-specific code outside an approved adapter and ADR.
11. Every behavior change needs acceptance criteria, tests, logs, failure behavior, and rollback notes.

## Claude Code Auto Operating Rules
Use Claude Code `auto` for routine coding and verification. Auto mode must be enabled in the developer's user setting (`~/.claude/settings.json`); the repository project setting only provides shared allow/ask/deny and hook policy. Work autonomously within an approved GitHub issue, but stop for human approval before:
- editing product scope, gateway hardware contract, or safety policy documents
- changing occupancy state semantics
- changing safety I/O mapping or stop-request behavior
- deploying or destroying AWS resources
- modifying IAM, certificates, network policies, or production data
- pushing, merging, tagging, or publishing releases
- destructive database migrations

Use project subagents for focused work. Use isolated worktrees for parallel implementation. After implementation, invoke a read-only reviewer and test author before presenting completion.

## Required Work Sequence
1. Read the issue and acceptance criteria.
2. Explore and write a short implementation plan.
3. Update versioned contracts first when interfaces change.
4. Add failing tests for domain and safety behavior.
5. Implement the smallest vertical slice.
6. Run `make verify-fast` during iteration.
7. Run `make verify` before PR.
8. Ask the safety reviewer for R2/R3 changes.
9. Update docs and changelog.
10. Prepare PR evidence; do not merge it yourself.

## Repository Boundaries
- `services/gateway-server`: Go edge runtime
- `services/cloud-backend`: TypeScript Lambda handlers and shared cloud domain
- `apps/frontend`: React application
- `infra/aws`: AWS CDK
- `contracts`: JSON Schema/OpenAPI/event topics
- `simulators`: camera and I/O simulators
- `tests`: integration, HIL definitions, acceptance evidence
- `packaging/gateway`: Ubuntu amd64 package and provisioning assets

## Command Contract
The repository must expose:
- `make setup`
- `make format`
- `make lint`
- `make typecheck`
- `make test`
- `make test-integration`
- `make test-contract`
- `make security`
- `make build`
- `make package-amd64`
- `make verify-fast`
- `make verify`

Never invent a command. Inspect the Makefile and package scripts first.

## Risk and Review
- R0 docs/tooling: T1 review or product owner
- R1 UI/report/non-safety API: T1 approval + CI
- R2 device, sync, auth, camera adapters, provisioning and update: T1 or T2 approval + integration evidence
- R3 occupancy, safety condition, I/O, stop request, restart interlock: both T1 and T2 approvals + HIL evidence

## Definition of Done
A change is done only when:
- acceptance criteria pass
- automated tests pass
- resource impact is measured where relevant
- failure and offline behavior are implemented
- user/operator/maintainer authorization is verified
- hardware portability remains intact
- logs and audit records are present
- rollback is documented
- required human tester evidence is attached
