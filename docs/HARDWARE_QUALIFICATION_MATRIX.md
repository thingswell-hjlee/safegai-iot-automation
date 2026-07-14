# SafeGAI Gateway Hardware Qualification Matrix v3.0

## 모델 정보

| 항목 | Reference IPC | Alternate IPC |
|---|---|---|
| 제조사/모델 |  |  |
| Revision |  |  |
| CPU |  |  |
| RAM |  |  |
| SSD |  |  |
| LAN Controller |  |  |
| BIOS Version |  |  |
| Ubuntu Image ID/Kernel |  |  |
| Gateway Package |  |  |
| Hardware Profile | ipc-lite-amd64-v1 | ipc-lite-amd64-v1 |

## 필수 적합성

| 시험 | 기준 | Reference | Alternate | 증거 |
|---|---|---|---|---|
| Ubuntu Autoinstall | 성공 |  |  |  |
| Dual LAN | 1Gbps Link |  |  |  |
| SSD Health | 정상 |  |  |  |
| AC Power Recovery | 20/20 |  |  |  |
| 4 Stream 8h | 중단 0 |  |  |  |
| Event Burst | 유실 0 |  |  |  |
| Alarm Latency | p95 ≤ 1s |  |  |  |
| DO Latency | p95 ≤ 500ms |  |  |  |
| CPU Average | ≤ 40% 목표 |  |  |  |
| CPU p95 | ≤ 70% |  |  |  |
| App Memory | ≤ 3GB |  |  |  |
| Cloud Outage | 72h |  |  |  |
| Outbox Replay | 중복/누락 0 |  |  |  |
| Update/Rollback | 성공 |  |  |  |
| Backup Restore | 성공 |  |  |  |
| IPC Replacement | Source 변경 0 |  |  |  |

## 판정
- `QUALIFIED`: 모든 필수항목 통과
- `CONDITIONAL`: 비안전 항목만 조건부 통과
- `REJECTED`: 안전·복구·호환성 항목 실패
