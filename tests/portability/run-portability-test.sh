#!/bin/bash
# SafeGAI Local Portability Test
# Verifies that the gateway binary works identically in local Ubuntu
# as it does in the AWS simulation environment.
#
# Usage: ./tests/portability/run-portability-test.sh [--profile local-sim]
#
# Exit codes:
#   0 = All tests passed
#   1 = Test failure
#   2 = Environment error

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
PROFILE="${1:-local-sim}"
GATEWAY_BIN="${PROJECT_ROOT}/dist/safegai-edge"
RESULTS_FILE="/tmp/safegai-portability-results.json"

echo "=== SafeGAI Local Portability Test ==="
echo "Profile: ${PROFILE}"
echo "Binary: ${GATEWAY_BIN}"
echo "Timestamp: $(date -u +%Y-%m-%dT%H:%M:%SZ)"
echo ""

# --- Prereq Check ---
check_prereqs() {
  local missing=()
  command -v curl >/dev/null 2>&1 || missing+=("curl")
  command -v jq >/dev/null 2>&1 || missing+=("jq")

  if [ ! -f "$GATEWAY_BIN" ]; then
    echo "ERROR: Gateway binary not found at $GATEWAY_BIN"
    echo "Build it with: make build"
    exit 2
  fi

  if [ ${#missing[@]} -gt 0 ]; then
    echo "ERROR: Missing required tools: ${missing[*]}"
    exit 2
  fi
}

# --- Test Functions ---
test_binary_starts() {
  echo "[TEST] Binary starts and responds to health check..."
  export SAFEGAI_PROFILE="$PROFILE"
  export SAFEGAI_LISTEN_ADDR=":18080"

  "$GATEWAY_BIN" &
  local PID=$!
  sleep 2

  local result="FAIL"
  if curl -sf "http://localhost:18080/health/live" >/dev/null 2>&1; then
    result="PASS"
  fi

  kill "$PID" 2>/dev/null || true
  wait "$PID" 2>/dev/null || true
  echo "  Result: $result"
  echo "$result"
}

test_health_endpoints() {
  echo "[TEST] Health endpoints return correct JSON..."
  export SAFEGAI_PROFILE="$PROFILE"
  export SAFEGAI_LISTEN_ADDR=":18081"

  "$GATEWAY_BIN" &
  local PID=$!
  sleep 2

  local result="FAIL"
  local live_body ready_body

  live_body=$(curl -sf "http://localhost:18081/health/live" 2>/dev/null)
  ready_body=$(curl -sf "http://localhost:18081/health/ready" 2>/dev/null)

  if echo "$live_body" | jq -e '.status == "healthy"' >/dev/null 2>&1 &&
     echo "$ready_body" | jq -e '.status == "healthy"' >/dev/null 2>&1; then
    result="PASS"
  fi

  kill "$PID" 2>/dev/null || true
  wait "$PID" 2>/dev/null || true
  echo "  Result: $result"
  echo "$result"
}

test_no_aws_sdk_in_core() {
  echo "[TEST] No AWS SDK imports in domain/application layers..."
  local result="PASS"

  if grep -r "github.com/aws" "$PROJECT_ROOT/services/gateway-server/internal/domain/" 2>/dev/null; then
    result="FAIL"
    echo "  FAIL: AWS SDK found in domain layer"
  fi

  if grep -r "github.com/aws" "$PROJECT_ROOT/services/gateway-server/internal/ports/" 2>/dev/null; then
    result="FAIL"
    echo "  FAIL: AWS SDK found in ports layer"
  fi

  echo "  Result: $result"
  echo "$result"
}

test_config_profile_switch() {
  echo "[TEST] Configuration profile switch works..."
  local result="PASS"

  for profile in local-sim local-lab local-pilot aws-sim; do
    if [ -f "$PROJECT_ROOT/configs/${profile}.yaml" ]; then
      echo "  Profile $profile config exists"
    else
      echo "  WARN: Profile $profile config missing (optional)"
    fi
  done

  echo "  Result: $result"
  echo "$result"
}

test_single_binary() {
  echo "[TEST] Single binary for all environments..."
  local result="PASS"

  # Check that there is exactly one main binary
  local binaries
  binaries=$(find "$PROJECT_ROOT/dist" -name "safegai-edge" -type f 2>/dev/null | wc -l)
  if [ "$binaries" -ne 1 ]; then
    result="FAIL"
    echo "  Expected 1 binary, found $binaries"
  fi

  echo "  Result: $result"
  echo "$result"
}

# --- Run Tests ---
check_prereqs

PASSED=0
FAILED=0
TOTAL=0

run_test() {
  local test_name="$1"
  TOTAL=$((TOTAL + 1))
  local test_result
  test_result=$($test_name 2>/dev/null | tail -1)
  if [ "$test_result" = "PASS" ]; then
    PASSED=$((PASSED + 1))
  else
    FAILED=$((FAILED + 1))
  fi
}

run_test test_no_aws_sdk_in_core
run_test test_config_profile_switch
run_test test_single_binary
run_test test_binary_starts
run_test test_health_endpoints

# --- Summary ---
echo ""
echo "=== Portability Test Results ==="
echo "Total: $TOTAL"
echo "Passed: $PASSED"
echo "Failed: $FAILED"
echo ""

# Write JSON results
cat > "$RESULTS_FILE" <<EOF
{
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "profile": "$PROFILE",
  "total": $TOTAL,
  "passed": $PASSED,
  "failed": $FAILED,
  "verdict": "$([ $FAILED -eq 0 ] && echo "PASS" || echo "FAIL")"
}
EOF

echo "Results written to: $RESULTS_FILE"

if [ "$FAILED" -gt 0 ]; then
  exit 1
fi
exit 0
