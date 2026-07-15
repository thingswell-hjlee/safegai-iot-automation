# PR Traceability Matrix

## Integration Branch: integration/mvp-consolidation

---

## PR #2 - Module Plan

| Field | Value |
|-------|-------|
| Branch | origin/feat/module-plan |
| Source Commit | 697e8fa |
| Integrated | Yes |
| Integration Commit | 94ec7c8 |
| Method | Cherry-pick (clean) |
| Files Included | docs/MODULE_PLAN.md, .agents/tasks/task-safegai-mvp-modules/** |
| Files Excluded | None |
| Exclusion Reason | N/A |
| Contract Modified | No |
| Test Result | N/A (documentation only) |
| Remaining Gap | None |

---

## PR #3 - M00 Foundation

| Field | Value |
|-------|-------|
| Branch | origin/feat/m00-foundation |
| Source Commit | a085eea |
| Integrated | Yes |
| Integration Commit | 0b5b478 |
| Method | Cherry-pick (clean) |
| Files Included | contracts/**, services/gateway-server/**, .gitignore, scripts/** |
| Files Excluded | None |
| Exclusion Reason | N/A |
| Contract Modified | Yes (post-integration: EquipmentState, SafetyDecision, ActuationCommand updated to canonical) |
| Test Result | All Go tests pass |
| Remaining Gap | None |

---

## PR #4 - M11 Quality Framework

| Field | Value |
|-------|-------|
| Branch | origin/feat/m11-quality-framework |
| Source Commit | 1e7cfc6 |
| Integrated | Yes (partial) |
| Integration Commit | 2210d52 (cherry-pick) + 0231ad2 (alignment) |
| Method | Cherry-pick with conflict resolution |
| Files Included | scripts/run-contract-tests.sh, tests/contract/**, tests/scenarios/**, tests/unit/README.md, Makefile |
| Files Excluded | contracts/api/local-api-v1.schema.json, contracts/errors/error-model-v1.schema.json, contracts/events/** (9 files) |
| Exclusion Reason | PR #3 canonical contracts preserved; PR #4 schemas had different structure and non-canonical enum values |
| Contract Modified | No (PR #3 contracts kept; PR #4 schemas discarded) |
| Test Result | 17 contract validations pass, 7 fixture tests pass |
| Remaining Gap | None |

---

## PR #5 - M01 Hybrid App Mock

| Field | Value |
|-------|-------|
| Branch | origin/feat/m01-hybrid-app-mock |
| Source Commit | 73ab255 |
| Integrated | Yes |
| Integration Commit | 5b22dbb |
| Method | Cherry-pick (clean) |
| Files Included | apps/frontend/**, .agents/tasks/**/FEAT-004.md |
| Files Excluded | None |
| Exclusion Reason | N/A |
| Contract Modified | No |
| Test Result | Not run (BLOCKED_NETWORK: npm install requires network) |
| Remaining Gap | Frontend type definitions reference local types that may diverge from canonical; needs TypeScript verification when npm is available |

---

## PR #6 - M02 Camera Adapter

| Field | Value |
|-------|-------|
| Branch | origin/feat/m02-camera-adapter |
| Source Commit | 628d7b1 |
| Integrated | Yes |
| Integration Commit | 3f4d8bd (cherry-pick) + 6f6cf92 (consolidation) |
| Method | Cherry-pick with envelope.go conflict resolution |
| Files Included | services/gateway-server/internal/adapters/camera/**, services/gateway-server/internal/domain/normalizer/**, simulators/camera/** |
| Files Excluded | services/gateway-server/internal/domain/events/envelope.go (theirs version) |
| Exclusion Reason | Canonical envelope from PR #3 retained; Payload field added separately |
| Contract Modified | Yes (added Payload []byte field to EventEnvelope) |
| Test Result | All Go tests pass (camera simulator, normalizer) |
| Remaining Gap | None |

---

## PR #7 - M03 Device/Sensor Input

| Field | Value |
|-------|-------|
| Branch | origin/feat/m03-device-sensor-input |
| Source Commit | 0db8bf2 |
| Integrated | Yes |
| Integration Commit | 92930bd (cherry-pick) + 6f6cf92 (consolidation) |
| Method | Cherry-pick with errors.go and types.go conflict resolution |
| Files Included | services/gateway-server/internal/adapters/io/**, services/gateway-server/internal/domain/equipment/**, simulators/io/** |
| Files Excluded | services/gateway-server/internal/domain/events/types.go (theirs), services/gateway-server/cmd/safegai-edge/main.go (theirs) |
| Exclusion Reason | Canonical types from PR #3 retained; theirs had duplicate definitions. Extended errors.go (theirs) was used as superset. |
| Contract Modified | Yes (EquipmentState updated: removed RESTART_REQUESTED, added STARTING/STOPPING/FAULT/OFFLINE) |
| Test Result | All Go tests pass (io simulator, equipment state) |
| Remaining Gap | None |

---

## PR #8 - M04 Zone State Engine

| Field | Value |
|-------|-------|
| Branch | origin/feat/m04-zone-state-engine |
| Source Commit | d9c9d22 |
| Integrated | Yes |
| Integration Commit | 9957e86 |
| Method | Cherry-pick with 4 conflicts resolved (ours kept for .gitignore, main.go, errors.go, types.go) |
| Files Included | services/gateway-server/internal/domain/occupancy/** |
| Files Excluded | Duplicate .gitignore, main.go, errors.go, types.go |
| Exclusion Reason | Canonical versions already present from earlier PRs |
| Contract Modified | No (occupancy module uses local types with matching canonical values) |
| Test Result | All Go tests pass (occupancy state machine, staleness, forbidden transitions) |
| Remaining Gap | Local OccupancyState type not yet refactored to reference events.OccupancyState |

---

## PR #9 - M05 Activity Engine

| Field | Value |
|-------|-------|
| Branch | origin/feat/m05-activity-engine |
| Source Commit | 6dd7838 |
| Integrated | Yes |
| Integration Commit | 77c48d3 |
| Method | Cherry-pick with 4 conflicts resolved (ours kept) |
| Files Included | services/gateway-server/internal/domain/safety/** |
| Files Excluded | Duplicate .gitignore, main.go, errors.go, types.go |
| Exclusion Reason | Canonical versions already present |
| Contract Modified | No (safety module uses local SafetyDecision type with matching canonical values) |
| Test Result | All Go tests pass (evaluator, rules, dedup, work window) |
| Remaining Gap | Local SafetyDecision type not yet refactored to reference events.SafetyDecision |

---

## PR #10 - M06 Output/Alarm

| Field | Value |
|-------|-------|
| Branch | origin/feat/m06-output-alarm |
| Source Commit | cac01e2 |
| Integrated | Yes |
| Integration Commit | eaeb87c |
| Method | Cherry-pick with 4 conflicts resolved (ours kept) |
| Files Included | services/gateway-server/internal/adapters/io/output/**, services/gateway-server/internal/domain/actuation/** |
| Files Excluded | Duplicate .gitignore, main.go, errors.go, types.go |
| Exclusion Reason | Canonical versions already present |
| Contract Modified | No (actuation uses local CommandType; values map to canonical at boundary) |
| Test Result | All Go tests pass (actuation service, dedup, replay guard, output adapter) |
| Remaining Gap | Internal command names (SIREN, STOP_REQUEST_PULSE, AUDIO_ANNOUNCEMENT) differ from external canonical names; adapter mapping needed |

---

## PR #11 - M08 Media Gateway

| Field | Value |
|-------|-------|
| Branch | origin/feat/m08-media-gateway |
| Source Commit | 8b65bbe |
| Integrated | Yes |
| Integration Commit | 441325c + 4be38b7 (consolidation) |
| Method | Cherry-pick with .gitignore and errors.go conflicts resolved |
| Files Included | services/gateway-server/internal/adapters/media/**, infra/edge/mediamtx/** |
| Files Excluded | Duplicate .gitignore, errors.go (CapacityLimitError merged manually) |
| Exclusion Reason | Canonical errors retained; CapacityLimitError added to existing |
| Contract Modified | No |
| Test Result | All Go tests pass (media config, manager) |
| Remaining Gap | None |

---

## PR #12 - M09 AWS Cloud

| Field | Value |
|-------|-------|
| Branch | origin/feat/m09-aws-cloud |
| Source Commit | 4147a03 |
| Integrated | Yes |
| Integration Commit | 4fb1fe5 |
| Method | Cherry-pick with .gitignore conflict resolved |
| Files Included | infra/aws/**, services/cloud-backend/**, contracts/mqtt/topics.md, .agents/tasks/**/FEAT-011.md |
| Files Excluded | .gitignore (merged) |
| Exclusion Reason | Combined .gitignore entries |
| Contract Modified | No |
| Test Result | Not run (TypeScript; npm install requires network) |
| Remaining Gap | TypeScript types in cloud-backend reference local definitions that need alignment with canonical contracts |

---

## PR #13 - M10 Operations/Maintenance

| Field | Value |
|-------|-------|
| Branch | origin/feat/m10-ops-maintenance |
| Source Commit | be1555b |
| Integrated | Yes |
| Integration Commit | cc4ebfd + 4be38b7 (consolidation) |
| Method | Cherry-pick with 5 conflicts resolved |
| Files Included | services/gateway-server/internal/auth/**, services/gateway-server/internal/cloud/outbox/**, services/gateway-server/internal/httpapi/**, services/gateway-server/internal/observability/**, services/gateway-server/internal/storage/**, services/gateway-server/cmd/safegai-edge/main.go |
| Files Excluded | Duplicate .gitignore, errors.go, envelope.go, types.go |
| Exclusion Reason | Canonical versions retained; model types (SafetyEvent, AuditEntry, etc.) added to canonical package |
| Contract Modified | Yes (added SafetyEvent, AuditEntry, OutboxItem, ConfigVersion, User to events package) |
| Test Result | All Go tests pass (auth, outbox, httpapi, observability, storage/memory) |
| Remaining Gap | None |

---

## Summary

| PR | Status | Conflicts | Contract Changes |
|----|--------|-----------|-----------------|
| #2 | Fully integrated | 0 | None |
| #3 | Fully integrated | 0 | Canonical baseline (updated post-integration) |
| #4 | Partially integrated | 9 (schemas) | Schemas excluded; tests aligned |
| #5 | Fully integrated | 0 | None |
| #6 | Fully integrated | 1 (envelope.go) | Payload field added |
| #7 | Fully integrated | 2 (errors.go, types.go) | EquipmentState canonical alignment |
| #8 | Fully integrated | 4 (standard duplicates) | None |
| #9 | Fully integrated | 4 (standard duplicates) | None |
| #10 | Fully integrated | 4 (standard duplicates) | None |
| #11 | Fully integrated | 2 (.gitignore, errors.go) | CapacityLimitError added |
| #12 | Fully integrated | 1 (.gitignore) | None |
| #13 | Fully integrated | 5 (all domain files) | Model types added |

**Total PRs integrated:** 12/12
**Total conflicts resolved:** 32
**Contract validations passing:** 24/24 (17 shell + 7 Python)
**Go test packages passing:** 15/15
**Blocked tests:** Frontend (npm), Cloud backend (npm) - BLOCKED_NETWORK
