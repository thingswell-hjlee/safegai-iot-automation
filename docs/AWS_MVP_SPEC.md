# SafeGAI AWS 최소연계 개발사양 v3.0

## 1. 목표
현장 안전을 제어하지 않고 Gateway 상태, Hardware Profile, 위험 이벤트, 대표 이미지, 알림, 원격 조회와 증빙 백업만 제공한다.

## 2. Region과 환경
- Region: `ap-northeast-2`
- Environment: `dev`, `pilot`
- Pilot 이후 `prod` 분리
- 개발자 노트북에서 Pilot 직접 Deploy 금지
- GitHub Actions OIDC Role로 배포

## 3. 확정 AWS 서비스

| 서비스 | 용도 |
|---|---|
| AWS IoT Core | Gateway MQTT/mTLS, Thing, Certificate, Policy |
| Named Device Shadows | `health`, `settings` 상태 |
| IoT Rules | Metadata Lambda 전달, Image S3 저장 |
| Lambda | `ingest-handler`, `admin-api-handler` |
| DynamoDB | `Gateways`, `Events` |
| S3 | Event Thumbnails, Cloud Frontend, Release Metadata |
| API Gateway HTTP API | Cloud Web REST API |
| Cognito User Pool | Operator/Maintainer Login |
| CloudFront | Cloud Web Delivery |
| SNS | Email and Optional SMS Notification |
| CloudWatch | Logs, Metrics, Alarms |
| AWS Budgets | Dev/Pilot Cost Alarm |
| CDK TypeScript | All Infrastructure |

Greengrass, ECS, EKS, RDS, OpenSearch, AppSync, Step Functions는 MVP에서 제외한다.

## 4. MQTT Topic

```text
safegai/v1/{tenant}/{site}/{gateway}/status
safegai/v1/{tenant}/{site}/{gateway}/events
safegai/v1/{tenant}/{site}/{gateway}/images/{eventId}
safegai/v1/{tenant}/{site}/{gateway}/acks
```

- QoS 1
- Event Metadata와 Image는 별도 Message
- Image Payload는 Raw JPEG Binary, 최대 96KB
- Base64 JSON Embedding 금지
- Topic Policy는 각 Gateway Identity로 제한
- Cloud-to-device Direct Actuator Command Topic은 존재하지 않음

## 5. Gateway Identity와 Hardware Inventory
각 Gateway는 다음을 등록한다.

- `gatewayId`
- `tenantId`, `siteId`
- X.509 Certificate ID
- `hardwareProfileId`: `ipc-lite-amd64-v1`
- Manufacturer, Model, Revision
- Serial Number
- Architecture: `amd64`
- CPU Model/Core Count
- Memory MiB
- SSD Model/Capacity/Health
- NIC Count/Driver/Link
- BIOS Version
- TPM/Watchdog Capability
- OS Image ID
- Kernel Version
- SafeGAI Package Version

Manufacturer와 Model은 운영·유지보수 정보이며 Cloud Logic의 기능분기 조건으로 사용하지 않는다.

## 6. Device Shadows

### `health`
Reported Only:
- Online/LastSeen
- Gateway Version
- Hardware Profile and OS Image ID
- Camera/I/O Summary
- CPU/RAM/Disk/Temperature
- SSD Health
- NIC Link State
- Outbox Depth

### `settings`
Desired/Reported Allowlist Only:
- Heartbeat Interval
- Cloud Thumbnail On/Off
- Log Level Expiration
- Notification Policy Version

Safety Rules, I/O Mapping, Stop-request Behavior, Restart Interlock, BIOS/OS Image는 Shadow로 변경하지 않는다.

## 7. Lambda

### 7.1 `ingest-handler`
Trigger: IoT Rule
- Schema Validation
- Tenant/Site/Gateway Identity Verification
- Hardware Profile Allowlist Validation
- Idempotent Event Put
- Gateways Last-seen/Health Update
- Notification Policy Evaluation
- SNS Publish
- Structured Log and Metric

### 7.2 `admin-api-handler`
Trigger: HTTP API
- Current Site/Gateway Status
- Hardware/OS/Version Inventory
- Event List/Detail
- Acknowledge/Resolve/Classification
- Basic Daily/Weekly/Monthly Aggregation
- Presigned S3 Thumbnail GET URL
- Qualified Hardware Model 조회
- Software Version Inventory

Machine-control Endpoint는 구현하지 않는다.

## 8. DynamoDB

### `Gateways`
- PK: `tenantId`
- SK: `siteId#gatewayId`
- Attributes: LastSeen, Status, Version, HardwareProfile, HardwareInventory, OSImageId, Health, Cameras, OutboxDepth
- GSI1: `gatewayId` for Certificate/Topic-bound Ingest Lookup

### `Events`
- PK: `tenantId#siteId`
- SK: `detectedAt#eventId`
- Attributes: GatewayId, CameraId, ZoneId, Severity, Occupancy, EquipmentState, Actions, Ack, Resolve, Classification, ImageKey
- GSI1: `gatewayId` + `detectedAt#eventId`
- Conditional Write on EventId/IdempotencyKey
- TTL Default 365 Days

## 9. S3
- Event Thumbnail Key: `{tenant}/{site}/{yyyy}/{mm}/{dd}/{eventId}.jpg`
- Public Access Block
- Encryption at Rest
- Lifecycle: 365 Days Default
- Frontend Bucket와 Evidence Bucket 분리
- Local Full Image는 자동 Upload하지 않음
- Release Metadata:
  - amd64 Package
  - Checksum/Signature/SBOM
  - Release Manifest
  - Supported Hardware Profile
  - Qualified Model List
  - OS Image ID

## 10. Cognito/RBAC

### Cloud Operator
- Cognito Group: `operator`
- Tenant/Site Claim 범위 내 Status/Events/Reports 조회
- Acknowledge/Resolve/Classify

### Cloud Maintainer
- Cognito Group: `maintainer`
- Operator Permissions
- View Diagnostics/Hardware/Version
- Request Diagnostic Package Generation Only

Cloud Maintainer는 안전 I/O Mapping 변경이나 현장출력 실행 권한이 없다.

## 11. Cloud Frontend
- React Static Build in S3 + CloudFront
- Cognito Login
- Responsive Mobile/Desktop
- No Live Video Relay
- No Direct Camera Credentials
- Event Thumbnail and Metadata Only
- Gateway Hardware/OS/SSD/NIC Health 화면
- Hardware Model은 표시정보이며 사용자 기능 차이를 만들지 않음

## 12. Release Manifest

```json
{
  "version": "0.5.0-rc.1",
  "architecture": "amd64",
  "package": "safegai-edge_0.5.0-rc.1_linux_amd64.deb",
  "hardwareProfiles": ["ipc-lite-amd64-v1"],
  "minimumMemoryMiB": 7600,
  "minimumStorageGiB": 120,
  "minimumEthernetPorts": 2,
  "osImageIds": ["ubuntu-24.04-amd64-sg-001"],
  "rollbackVersion": "0.4.2"
}
```

MVP Cloud는 Update 가능 여부를 표시하지만 자동 설치 명령을 현장으로 전송하지 않는다.

## 13. CDK Layout

```text
infra/aws/
├─ bin/app.ts
├─ lib/foundation-stack.ts
├─ lib/iot-stack.ts
├─ lib/data-stack.ts
├─ lib/api-stack.ts
├─ lib/web-stack.ts
├─ config/dev.ts
├─ config/pilot.ts
└─ test/
```

## 14. CI/CD
- PR: Test, Lint, `cdk synth`, Policy Checks
- Main: Dev Auto Deploy through GitHub OIDC
- Pilot: Version Tag + Environment Manual Approval
- `cdk diff` Artifact Required
- Destructive Replacement Blocks Deployment
- No Long-lived AWS Access Key in GitHub
- Edge Package Release와 AWS Deploy는 같은 Release Manifest를 참조

## 15. Alarm
- Gateway Offline > 3 Minutes
- Hardware Profile Mismatch
- SSD Warning/Critical
- NIC Camera LAN Down
- Ingest Error Rate
- Lambda Error/Throttle
- DynamoDB Throttle
- S3 Rule Failure
- Budget Threshold
- Pilot Deploy Smoke-test Failure

## 16. AWS MVP 보안 경계
- Gateway는 X.509 인증서와 Topic-scoped IoT Policy만 사용한다.
- 웹 사용자는 Cognito JWT의 Tenant/Site Claim을 Lambda에서 다시 검증한다.
- API Gateway 또는 Frontend의 표시상 권한만 신뢰하지 않는다.
- Pilot 배포는 GitHub OIDC의 단기 자격증명과 승인된 Environment를 통해서만 수행한다.
- AWS에서 Gateway의 DO·정지요청·재가동 신호를 직접 실행하는 API와 Topic은 만들지 않는다.
- Hardware Inventory를 원격 명령 실행의 근거로 사용하지 않는다.
