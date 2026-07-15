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
EXPECTED_STATES='["OCCUPIED", "STALE", "UNKNOWN", "VACANT_CONFIRMED", "VACANT_PENDING"]'

if [ -f "$OCCUPANCY_SCHEMA" ]; then
  actual_states=$(python3 -c "
import json, sys
with open('$OCCUPANCY_SCHEMA') as f:
    schema = json.load(f)
# Canonical schema uses \$defs.OccupancyState.enum
states = schema.get('\$defs', {}).get('OccupancyState', {}).get('enum', [])
if not states:
    states = schema.get('properties', {}).get('currentState', {}).get('enum', [])
print(json.dumps(sorted(states)))
")

  if [ "$actual_states" = "$EXPECTED_STATES" ]; then
    pass "occupancy-state enum contains exactly: OCCUPIED, VACANT_PENDING, VACANT_CONFIRMED, UNKNOWN, STALE"
  else
    fail "occupancy-state enum" "expected $EXPECTED_STATES but got $actual_states"
  fi

  # Verify VACANT_CONFIRMED is the only valid vacancy state
  has_vc=$(python3 -c "
import json, sys
with open('$OCCUPANCY_SCHEMA') as f:
    schema = json.load(f)
states = schema.get('\$defs', {}).get('OccupancyState', {}).get('enum', [])
sys.exit(0 if 'VACANT_CONFIRMED' in states else 1)
" && echo "yes" || echo "no")
  if [ "$has_vc" = "yes" ]; then
    pass "VACANT_CONFIRMED is present as the valid vacancy state"
  else
    fail "VACANT_CONFIRMED" "not found in occupancy state enum"
  fi
else
  fail "occupancy-state schema" "file not found at $OCCUPANCY_SCHEMA"
fi
echo ""

# Step 4: Validate safety-decision schema contains canonical decision types
echo "--- Safety Decision Enum Validation ---"
SAFETY_SCHEMA="$PROJECT_ROOT/contracts/events/safety-decision-v1.schema.json"
EXPECTED_DECISIONS='["MAINTENANCE_MONITORING", "RESTART_INTERLOCK", "SAFE", "SAFETY_CONFIRMATION_UNAVAILABLE", "STOP_REQUEST_REQUIRED", "WARNING"]'

if [ -f "$SAFETY_SCHEMA" ]; then
  actual_decisions=$(python3 -c "
import json, sys
with open('$SAFETY_SCHEMA') as f:
    schema = json.load(f)
# Canonical schema uses \$defs.SafetyDecision.enum
decisions = schema.get('\$defs', {}).get('SafetyDecision', {}).get('enum', [])
if not decisions:
    decisions = schema.get('properties', {}).get('decision', {}).get('enum', [])
print(json.dumps(sorted(decisions)))
")

  if [ "$actual_decisions" = "$EXPECTED_DECISIONS" ]; then
    pass "safety-decision enum contains canonical decision types"
  else
    fail "safety-decision enum" "expected $EXPECTED_DECISIONS but got $actual_decisions"
  fi

  # Verify banned values are NOT present
  banned_check=$(python3 -c "
import json, sys
with open('$SAFETY_SCHEMA') as f:
    schema = json.load(f)
decisions = schema.get('\$defs', {}).get('SafetyDecision', {}).get('enum', [])
banned = {'OPERATION_PERMITTED', 'ALLOW_START', 'EMERGENCY_STOP'}
found = banned & set(decisions)
if found:
    print(','.join(found))
    sys.exit(1)
sys.exit(0)
" && echo "" || echo "found")
  if [ -z "$banned_check" ]; then
    pass "No banned safety decision values present"
  else
    fail "safety-decision banned" "found banned values in enum"
  fi
else
  fail "safety-decision schema" "file not found at $SAFETY_SCHEMA"
fi
echo ""

# Step 5: Validate equipment-state schema
echo "--- Equipment State Enum Validation ---"
EQUIPMENT_SCHEMA="$PROJECT_ROOT/contracts/events/equipment-state-v1.schema.json"
EXPECTED_EQUIPMENT='["FAULT", "OFFLINE", "RUNNING", "STARTING", "STOPPED", "STOPPING", "UNKNOWN"]'

if [ -f "$EQUIPMENT_SCHEMA" ]; then
  actual_equipment=$(python3 -c "
import json, sys
with open('$EQUIPMENT_SCHEMA') as f:
    schema = json.load(f)
states = schema.get('\$defs', {}).get('EquipmentState', {}).get('enum', [])
print(json.dumps(sorted(states)))
")

  if [ "$actual_equipment" = "$EXPECTED_EQUIPMENT" ]; then
    pass "equipment-state enum contains canonical states"
  else
    fail "equipment-state enum" "expected $EXPECTED_EQUIPMENT but got $actual_equipment"
  fi

  # Verify RESTART_REQUESTED is NOT present
  restart_check=$(python3 -c "
import json, sys
with open('$EQUIPMENT_SCHEMA') as f:
    schema = json.load(f)
states = schema.get('\$defs', {}).get('EquipmentState', {}).get('enum', [])
sys.exit(1 if 'RESTART_REQUESTED' in states else 0)
" && echo "" || echo "found")
  if [ -z "$restart_check" ]; then
    pass "RESTART_REQUESTED is not in EquipmentState (correct)"
  else
    fail "equipment-state" "RESTART_REQUESTED must not be an EquipmentState"
  fi
else
  fail "equipment-state schema" "file not found at $EQUIPMENT_SCHEMA"
fi
echo ""

# Step 6: Run Python contract validator for fixture validation
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
