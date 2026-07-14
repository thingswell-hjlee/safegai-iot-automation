#!/usr/bin/env bash
# Contract validation script for SafeGAI IoT platform.
# Validates JSON schema files are well-formed and contain expected domain values.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

PASS=0
FAIL=0
ERRORS=()

pass() {
  PASS=$((PASS + 1))
  printf '  \033[32mPASS\033[0m %s\n' "$1"
}

fail() {
  FAIL=$((FAIL + 1))
  ERRORS+=("$1: $2")
  printf '  \033[31mFAIL\033[0m %s - %s\n' "$1" "$2"
}

echo "=== SafeGAI Contract Validation ==="
echo ""

# Step 1: JSON Syntax validation (delegates to check-json.py)
echo "--- JSON Syntax Check ---"
if python3 "$SCRIPT_DIR/check-json.py"; then
  pass "All JSON files have valid syntax"
else
  fail "JSON syntax" "One or more files have invalid JSON"
fi
echo ""

# Step 2: Validate all schema files have required meta-fields
echo "--- Schema Meta-Field Validation ---"
schema_files=$(find "$PROJECT_ROOT/contracts" -name '*.schema.json' -type f | sort)

for schema_file in $schema_files; do
  rel_path="${schema_file#$PROJECT_ROOT/}"
  
  # Check for required fields: $schema, $id, title, type
  missing=""
  for field in '$schema' '$id' 'title' 'type'; do
    key=$(echo "$field" | sed 's/\$//')
    if [ "$field" = '$schema' ]; then
      if ! python3 -c "import json,sys; d=json.load(open('$schema_file')); sys.exit(0 if '\$schema' in d else 1)" 2>/dev/null; then
        missing="$missing \$schema"
      fi
    elif [ "$field" = '$id' ]; then
      if ! python3 -c "import json,sys; d=json.load(open('$schema_file')); sys.exit(0 if '\$id' in d else 1)" 2>/dev/null; then
        missing="$missing \$id"
      fi
    else
      if ! python3 -c "import json,sys; d=json.load(open('$schema_file')); sys.exit(0 if '$field' in d else 1)" 2>/dev/null; then
        missing="$missing $field"
      fi
    fi
  done
  
  if [ -z "$missing" ]; then
    pass "$rel_path has all required meta-fields"
  else
    fail "$rel_path" "missing meta-fields:$missing"
  fi
done
echo ""

# Step 3: Validate occupancy-state schema contains exactly the expected enum values
echo "--- Occupancy State Enum Validation ---"
OCCUPANCY_SCHEMA="$PROJECT_ROOT/contracts/events/occupancy-state-v1.schema.json"
EXPECTED_STATES='["OCCUPIED", "VACANT_PENDING", "VACANT_CONFIRMED", "UNKNOWN", "STALE"]'

if [ -f "$OCCUPANCY_SCHEMA" ]; then
  actual_states=$(python3 -c "
import json, sys
with open('$OCCUPANCY_SCHEMA') as f:
    schema = json.load(f)
states = schema.get('properties', {}).get('state', {}).get('enum', [])
# Sort for comparison
print(json.dumps(sorted(states)))
")
  expected_sorted=$(python3 -c "import json; print(json.dumps(sorted($EXPECTED_STATES)))")
  
  if [ "$actual_states" = "$expected_sorted" ]; then
    pass "occupancy-state enum contains exactly: OCCUPIED, VACANT_PENDING, VACANT_CONFIRMED, UNKNOWN, STALE"
  else
    fail "occupancy-state enum" "expected $expected_sorted but got $actual_states"
  fi
  
  # Verify VACANT_CONFIRMED is the only valid vacancy state (it must be in the enum)
  has_vc=$(python3 -c "
import json, sys
with open('$OCCUPANCY_SCHEMA') as f:
    schema = json.load(f)
states = schema.get('properties', {}).get('state', {}).get('enum', [])
sys.exit(0 if 'VACANT_CONFIRMED' in states else 1)
")
  if [ $? -eq 0 ]; then
    pass "VACANT_CONFIRMED is present as the valid vacancy state"
  else
    fail "VACANT_CONFIRMED" "not found in occupancy state enum"
  fi
else
  fail "occupancy-state schema" "file not found at $OCCUPANCY_SCHEMA"
fi
echo ""

# Step 4: Validate safety-decision schema contains expected decision types
echo "--- Safety Decision Enum Validation ---"
SAFETY_SCHEMA="$PROJECT_ROOT/contracts/events/safety-decision-v1.schema.json"
EXPECTED_DECISIONS='["STOP_REQUEST_REQUIRED", "OPERATION_PERMITTED", "HOLD_CURRENT_STATE", "EMERGENCY_STOP"]'

if [ -f "$SAFETY_SCHEMA" ]; then
  actual_decisions=$(python3 -c "
import json, sys
with open('$SAFETY_SCHEMA') as f:
    schema = json.load(f)
decisions = schema.get('properties', {}).get('decision', {}).get('enum', [])
print(json.dumps(sorted(decisions)))
")
  expected_sorted=$(python3 -c "import json; print(json.dumps(sorted($EXPECTED_DECISIONS)))")
  
  if [ "$actual_decisions" = "$expected_sorted" ]; then
    pass "safety-decision enum contains expected decision types"
  else
    fail "safety-decision enum" "expected $expected_sorted but got $actual_decisions"
  fi
else
  fail "safety-decision schema" "file not found at $SAFETY_SCHEMA"
fi
echo ""

# Step 5: Run Python contract validator for fixture validation
echo "--- Fixture Validation ---"
VALIDATOR="$PROJECT_ROOT/tests/contract/validate_contracts.py"
if [ -f "$VALIDATOR" ]; then
  if python3 "$VALIDATOR"; then
    pass "All fixture validations passed"
  else
    fail "fixture validation" "Python validator reported failures"
  fi
else
  fail "fixture validation" "validate_contracts.py not found"
fi
echo ""

# Summary
echo "=== Contract Validation Summary ==="
echo "  Passed: $PASS"
echo "  Failed: $FAIL"

if [ ${#ERRORS[@]} -gt 0 ]; then
  echo ""
  echo "  Failures:"
  for err in "${ERRORS[@]}"; do
    echo "    - $err"
  done
fi

echo ""
if [ $FAIL -gt 0 ]; then
  echo "CONTRACT VALIDATION FAILED"
  exit 1
else
  echo "CONTRACT VALIDATION PASSED"
  exit 0
fi
