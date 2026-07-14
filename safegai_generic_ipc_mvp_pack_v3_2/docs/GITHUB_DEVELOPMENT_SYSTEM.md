# SafeGAI 3인 팀 GitHub 개발체계 v3.0

## 1. 저장소
Private Monorepo 하나를 사용한다.

```text
safegai-platform/
├─ SOUL.md
├─ CLAUDE.md
├─ AIDLC.md
├─ AGENTS.md
├─ contracts/
├─ services/gateway-server/
├─ services/cloud-backend/
├─ apps/frontend/
├─ infra/
│  ├─ aws/
│  └─ edge/
│     ├─ autoinstall/
│     ├─ hardware-profiles/
│     ├─ netplan/
│     ├─ systemd/
│     └─ packaging/
├─ simulators/
├─ tests/
├─ docs/
├─ .claude/
├─ .kiro/
└─ .github/
```

MVP 중 Polyrepo 분리는 금지한다.

## 2. Branch
1인 개발에 `develop` Branch는 사용하지 않는다.

- `main`: 항상 배포 가능
- `feature/<issue>-<slug>`: 1~2일 이내
- `fix/<issue>-<slug>`
- `rc/<version>`: 현장 RC 안정화가 필요한 경우만
- `spec/<issue>-<slug>`: Kiro Specification 전용

Merge는 Squash를 기본으로 한다.

## 3. GitHub Project
Status:
- Backlog
- Ready
- In Development
- Functional Test
- HIL / Hardware Qualification
- Ready to Release
- Done

Issue 필수필드:
- Customer Value
- Role/Mode Affected
- Acceptance Criteria
- Risk R0-R3
- Hardware Dependency: None/Profile/Model
- Test Owner T1/T2
- Rollback

## 4. PR 크기
- 1 Issue, 1 Purpose
- Generated File 제외 400 Changed Lines 권장
- 800 Lines 초과 시 분할 또는 사유 기록
- Schema/Contract 변경은 소비자 코드와 같은 PR 또는 Versioned Migration PR
- Hardware Model 이름이 Source Logic에 추가되면 Review Block

## 5. 승인
- R0: T1 또는 제품책임자
- R1: T1
- R2: T1 또는 T2
- R3: T1 + T2
- D1은 R3 최종승인 불가
- `hardware-baseline` Label은 T2와 제품책임자 승인 필요

## 6. Main Ruleset
- Pull Request Required
- Required Status Checks
- Conversation Resolution
- No Direct/Force Push
- No Delete
- Signed/Verified Release Tag
- R3 Label PR은 두 Reviewer 요구
- Release Workflow와 Hardware Profile 변경은 CODEOWNERS 요구

## 7. CODEOWNERS 권장

```text
/SOUL.md                         @product-owner
/AIDLC.md                        @product-owner
/docs/PRODUCT_MVP_SPEC.md        @product-owner
/docs/GATEWAY_PRODUCT_SPEC.md    @product-owner @t2-tester
/docs/HARDWARE_QUALIFICATION_SPEC.md @t2-tester
/contracts/safety/               @t1-tester @t2-tester
/services/gateway-server/internal/safety/ @t1-tester @t2-tester
/infra/edge/hardware-profiles/   @t2-tester
/infra/aws/config/pilot*         @product-owner
/.github/workflows/release*      @product-owner @t2-tester
```

실제 GitHub 사용자명에 맞게 변경한다.

## 8. Actions

### `pr-ci`
- Changed Path Detection
- Go Format/Vet/Test
- TypeScript Lint/Typecheck/Test
- Contract Validation
- Secret Scan
- Dependency/SAST
- Native linux/amd64 Build
- amd64 Package Dry-run
- Frontend Build
- Ubuntu Autoinstall/Netplan/Systemd Static Validation
- Hardware Profile Schema Validation
- CDK Synth

### `main-dev-deploy`
- Main Merge 후 AWS Dev 자동배포
- GitHub OIDC
- Smoke Test
- Gateway Package는 Artifact만 생성하고 실제 IPC 자동설치 금지

### `nightly-integration`
- Camera Simulator
- I/O Simulator
- Gateway/Cloud/Frontend E2E
- Offline/Replay
- Package Install/Upgrade/Rollback VM Test

### `hil-test`
- `workflow_dispatch`
- Dedicated QA Self-hosted Runner
- Reference 또는 Alternate IPC에 amd64 Package Deploy
- Latency/Resource/Failure Test
- Artifact Upload
- 외부 Fork PR 실행 금지

### `hardware-qualification`
- `workflow_dispatch`
- 입력: Model ID, Hardware Profile, OS Image ID
- `qualify-hardware.sh` 실행
- Test Evidence와 승인 Checklist 업로드
- Production AWS Credential 없음

### `release-pilot`
- Tag Trigger
- SBOM/Checksum/Signature
- amd64 `.deb`
- Release Manifest with Hardware Profile and OS Image ID
- CDK Diff Artifact
- Protected Pilot Environment Approval
- AWS Deploy and Smoke Test

## 9. AWS Authentication
GitHub Actions는 OIDC로 단기 AWS Credential을 발급한다.
- Long-lived AWS Access Key Secret 금지
- Dev Role과 Pilot Role 분리
- Pilot Role은 Protected Environment에서만 Assume

## 10. Release
- `v0.1.0-alpha`: Simulator Vertical Slice
- `v0.5.0-rc.1`: Full Testbed RC
- `v0.9.0-pilot.1`: First Site
- `v1.0.0`: 30-day Pilot Exit

Release Assets:
- `safegai-edge_<version>_linux_amd64.deb`
- Checksum/Signature
- SBOM
- Release Manifest
- Supported Hardware Profile
- Qualified Model List
- OS Image ID
- DB Migration Notes
- Rollback Package
- T1/T2 Evidence Links

## 11. Hardware Profile Contract
`infra/edge/hardware-profiles/schema.json`을 계약으로 관리한다.

필수 CI 검사:
- 모든 YAML이 Schema를 통과
- `profileId` 중복 금지
- Source Code에 Vendor Name 분기 금지
- Release Manifest가 승인 Profile을 참조
- Qualified Model은 T2 Evidence ID를 포함

## 12. AI 기록
PR에 다음을 기록한다.
- Claude Code/ChatGPT/Kiro 사용영역
- AI 생성 코드의 인간검증 내용
- Subagent Review 결과
- Test Commands and Evidence
- Hardware 영향과 Profile 변경 여부

AI Chat Transcript 자체는 공식 설계·승인 기록이 아니다.
