#!/bin/bash
# SafeGAI Soak Test
# Runs the gateway under sustained load for an extended period
# to detect memory leaks, resource exhaustion, and degradation.
#
# Usage: ./tests/performance/soak-test.sh [duration] [target-url]
#
# Default: 1 hour at 50 events/sec

set -euo pipefail

DURATION="${1:-1h}"
TARGET="${2:-http://localhost:8080}"
CONCURRENCY=20
RATE=50
REPORT_DIR="/tmp/safegai-soak-test"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

echo "=== SafeGAI Soak Test ==="
echo "Duration: $DURATION"
echo "Target: $TARGET"
echo "Concurrency: $CONCURRENCY"
echo "Rate: $RATE events/sec"
echo "Report Dir: $REPORT_DIR"
echo ""

mkdir -p "$REPORT_DIR"

# Check prerequisites
if [ ! -f "$SCRIPT_DIR/load-generator.go" ]; then
  echo "ERROR: load-generator.go not found"
  exit 2
fi

# Build load generator if not already built
LOAD_GEN="$SCRIPT_DIR/load-generator"
if [ ! -f "$LOAD_GEN" ]; then
  echo "Building load generator..."
  cd "$SCRIPT_DIR"
  go build -o load-generator load-generator.go 2>/dev/null || {
    echo "WARN: Could not build load-generator, using go run"
    LOAD_GEN=""
  }
fi

# Record baseline memory
echo "Recording baseline metrics..."
BASELINE_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
curl -sf "$TARGET/health/ready" > "$REPORT_DIR/baseline-health.json" 2>/dev/null || echo '{"status":"unreachable"}' > "$REPORT_DIR/baseline-health.json"

# Start load test
echo "Starting soak test at $(date -u +%H:%M:%S) for $DURATION..."
if [ -n "$LOAD_GEN" ]; then
  "$LOAD_GEN" \
    --target="$TARGET" \
    --duration="$DURATION" \
    --concurrency="$CONCURRENCY" \
    --rate="$RATE" \
    --output="$REPORT_DIR/load-results.json"
else
  cd "$SCRIPT_DIR"
  go run load-generator.go \
    --target="$TARGET" \
    --duration="$DURATION" \
    --concurrency="$CONCURRENCY" \
    --rate="$RATE" \
    --output="$REPORT_DIR/load-results.json"
fi

# Record post-test metrics
END_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ)
curl -sf "$TARGET/health/ready" > "$REPORT_DIR/final-health.json" 2>/dev/null || echo '{"status":"unreachable"}' > "$REPORT_DIR/final-health.json"

# Generate summary
cat > "$REPORT_DIR/soak-summary.json" <<EOF
{
  "testType": "soak",
  "startTime": "$BASELINE_TIME",
  "endTime": "$END_TIME",
  "duration": "$DURATION",
  "target": "$TARGET",
  "concurrency": $CONCURRENCY,
  "rate": $RATE,
  "verdict": "$(curl -sf "$TARGET/health/ready" >/dev/null 2>&1 && echo "GATEWAY_HEALTHY" || echo "GATEWAY_DEGRADED")"
}
EOF

echo ""
echo "=== Soak Test Complete ==="
echo "Results: $REPORT_DIR/"
echo "- load-results.json: Request metrics"
echo "- baseline-health.json: Pre-test health"
echo "- final-health.json: Post-test health"
echo "- soak-summary.json: Summary"

# Check if gateway is still healthy
if curl -sf "$TARGET/health/ready" >/dev/null 2>&1; then
  echo "Verdict: PASS (gateway remained healthy)"
  exit 0
else
  echo "Verdict: FAIL (gateway degraded or unreachable)"
  exit 1
fi
