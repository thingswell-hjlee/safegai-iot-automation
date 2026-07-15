# AWS-First Edge-Ready Architecture

## Design Principle

"AWS first, edge ready" means we develop and validate in the cloud, then deploy
the exact same binary to the factory floor. The architecture enforces this through
strict layering rules.

## Layer Diagram

```
+-------------------------------------------+
|         Configuration Layer               |
|  (Profile YAML: aws-sim, local-pilot)     |
+-------------------------------------------+
|         Adapter Layer (ports impl)        |
|  SimCamera | GenericHttp | VendorCamera   |
|  SimSensor | ModbusSensor                 |
|  SimOutput | ModbusTcpOutput              |
|  SimMedia  | MediaMTXAdapter              |
|  Disabled  | AwsIoTCloud                  |
+-------------------------------------------+
|         Application Layer                 |
|  Event Pipeline | Safety Engine | Outbox  |
|  Audit | Auth | HTTP API | Health         |
+-------------------------------------------+
|         Domain Layer                      |
|  Occupancy | Equipment | Safety Rules     |
|  Events | Normalizer | Actuation          |
+-------------------------------------------+
|         Storage Layer                     |
|  SQLite WAL (single file, portable)       |
+-------------------------------------------+
```

## Key Invariants

1. **Domain and Application layers have zero cloud imports**
2. **Safety rules are immutable across environments**
3. **Adapter selection is config-only** (no code changes)
4. **Single compiled binary** for all target environments
5. **Offline operation** is the default (cloud is optional)

## Component Architecture

### Gateway Server
```
services/gateway-server/
  cmd/safegai-edge/        Main binary entry point
  internal/
    ports/                 Interface definitions (9 ports)
    domain/                Business logic (no external deps)
      occupancy/           Zone state machine
      equipment/           Equipment state tracking
      safety/              Rule evaluation engine
      actuation/           Output decision logic
      events/              Event types and envelope
      normalizer/          Dedup and normalization
    adapters/              Port implementations
      camera/              Camera adapters
      io/                  Sensor/Output adapters
      media/               Media stream adapters
    config/                Profile-based configuration
    storage/               SQLite + memory stores
    auth/                  Session management
    httpapi/               REST API handlers
    cloud/                 Cloud sync (outbox pattern)
    observability/         Health and metrics
```

### Simulators
```
services/gateway-server/cmd/
  safegai-camera-sim/      4 cameras, 8 zones, HTTP events
  safegai-sensor-sim/      6 sensor types, periodic readings
  safegai-equipment-sim/   6 units, state machine transitions
  safegai-output-sim/      5 output types, command execution
  safegai-modbus-sim/      8DI/8DO Modbus TCP server
  safegai-scenario-runner/ S01-S14 orchestration
```

### AWS Infrastructure
```
infra/sim/
  bin/sim-app.ts           CDK app entry point
  lib/
    sim-network-stack.ts   VPC, subnets, security groups
    sim-identity-stack.ts  IAM roles, instance profile
    sim-data-stack.ts      DynamoDB, S3
    sim-iot-stack.ts       IoT Core thing, rules
    sim-gateway-stack.ts   EC2 instance with user-data
    sim-api-stack.ts       API Gateway, Lambda, Cognito
    sim-frontend-stack.ts  S3 + CloudFront SPA
    sim-observability-stack.ts  Dashboard, auto-stop
```

## Data Flow

```
Camera/Sensor/Modbus
        |
        v (adapter)
  Event Normalizer
        |
        v
  Safety Rule Engine --> Output Adapter --> Physical Device
        |
        v
  Event Store (SQLite)
        |
        v (outbox sync)
  Cloud (IoT Core) --> DynamoDB / S3
        |
        v
  API Gateway --> Frontend Dashboard
```

## Safety Architecture

```
Input Event --> Occupancy State Machine --> Rule Evaluation
                                                |
                        +----------+------------|----------+
                        |          |            |          |
                      SAFE      WARNING    STOP_REQ    INTERLOCK
                        |          |            |          |
                      (log)   (warn light)  (PLC stop)  (block restart)
                               (siren)
                               (voice)
```

Safety invariants:
- UNKNOWN/STALE zones are never treated as vacant
- STOP_REQUEST targets PLC/Safety Relay only, never direct machine power
- All safety decisions are audit-logged with full traceability
- Safety rules are fixed and cannot be modified at runtime

## Deployment Environments

| Environment | Profile | Adapters | Cloud |
|-------------|---------|----------|-------|
| AWS Sim Dev | aws-sim | All simulators | IoT Core |
| Local Sim | local-sim | All simulators | Disabled |
| Local Lab | local-lab | Real camera, sim others | Disabled |
| Local Pilot | local-pilot | All real hardware | Optional |

## Network Architecture (AWS Sim)

```
Internet
    |
CloudFront (Frontend)
    |
API Gateway + Cognito
    |
VPC (10.100.0.0/16)
  +-- Public Subnet AZ-a
  |     EC2 (Gateway + Sims)
  +-- Public Subnet AZ-b
        (failover capacity)
```

No NAT gateway (cost optimization). All traffic via public IPs.
SSM for instance management (no SSH keys).

## Cost Model

| Resource | Monthly Estimate | Note |
|----------|-----------------|------|
| EC2 t3.medium | ~$30 (8h/day) | Auto-stop saves 67% |
| DynamoDB | ~$5 | On-demand, low volume |
| S3 | ~$1 | 7-day retention |
| IoT Core | ~$1 | < 100K messages |
| CloudFront | ~$1 | Light traffic |
| API Gateway | ~$1 | < 10K requests |
| CloudWatch | ~$3 | Dashboard + logs |
| **Total** | **~$42/month** | With auto-stop |
