# AWS Simulation Performance Report

## Summary

| Metric | Target | Measured | Status |
|--------|--------|----------|--------|
| Event Throughput | >= 100 events/sec | (Pending) | - |
| P50 Latency | < 50ms | (Pending) | - |
| P99 Latency | < 200ms | (Pending) | - |
| Error Rate | < 0.1% | (Pending) | - |
| Memory Growth (1h) | < 50MB | (Pending) | - |
| CPU Usage (steady) | < 30% (t3.medium) | (Pending) | - |

## Test Configuration

| Parameter | Value |
|-----------|-------|
| Instance Type | t3.medium (2 vCPU, 4 GB RAM) |
| OS | Amazon Linux 2023 |
| Duration | 1 hour soak test |
| Concurrency | 20 workers |
| Target Rate | 100 events/sec |
| Event Types | Camera, Sensor, Equipment mixed |

## Throughput Test

- **Objective**: Verify gateway processes >= 100 events/sec sustained
- **Method**: Load generator sending mixed events for 5 minutes
- **Result**: (Pending execution)
- **Observations**: (Pending)

## Latency Test

- **Objective**: Verify sub-200ms P99 event processing latency
- **Method**: Health endpoint response time measurement
- **Result**: (Pending execution)
- **Observations**: (Pending)

## Soak Test

- **Objective**: Verify no memory leaks or degradation over 1 hour
- **Method**: Sustained 50 events/sec with memory monitoring
- **Result**: (Pending execution)
- **Observations**: (Pending)

## Resource Utilization

### CPU
- Baseline (idle): ~1-2%
- Under load (100 events/sec): (Pending)
- Peak: (Pending)

### Memory
- Baseline: ~20MB RSS
- After 1 hour load: (Pending)
- Growth rate: (Pending)

### Disk I/O
- SQLite writes: (Pending)
- WAL checkpoints: (Pending)

### Network
- Events ingested: (Pending)
- Cloud sync outbound: (Pending)

## Bottleneck Analysis

Based on architecture:
1. SQLite WAL writes - single-writer with WAL provides good throughput
2. Event processing pipeline - in-memory channel buffering
3. Safety rule evaluation - stateless, O(1) per event
4. Output command execution - bounded by adapter response time

## Recommendations

1. Monitor SQLite WAL size during sustained load
2. Consider batch processing for high-volume sensor data
3. Tune channel buffer sizes if drops observed
4. Set up CloudWatch custom metrics for production monitoring

## How to Run

```bash
# Build load generator
cd tests/performance
go build -o load-generator load-generator.go

# Quick test (5 min)
./load-generator --target=http://<gateway-ip>:8080 --duration=5m --concurrency=10 --rate=100

# Soak test (1 hour)
./soak-test.sh 1h http://<gateway-ip>:8080
```

## Evidence

Report generated: (Pending execution)
Report location: `evidence/aws-edge-ready/performance/`
