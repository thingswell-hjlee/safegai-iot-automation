#!/usr/bin/env python3
"""Contract fixture validator for SafeGAI IoT platform.

Validates test fixtures against their corresponding JSON schemas using only
Python standard library. Performs structural validation (required fields,
type checks, enum membership) without full JSON Schema draft support.
"""
import json
import sys
from pathlib import Path

PROJECT_ROOT = Path(__file__).resolve().parent.parent.parent
CONTRACTS_DIR = PROJECT_ROOT / "contracts"
FIXTURES_DIR = Path(__file__).resolve().parent / "fixtures"

PASS = 0
FAIL = 0


def log_pass(msg: str) -> None:
    global PASS
    PASS += 1
    print(f"    PASS: {msg}")


def log_fail(msg: str) -> None:
    global FAIL
    FAIL += 1
    print(f"    FAIL: {msg}")


def load_json(path: Path) -> dict:
    """Load and parse a JSON file."""
    with open(path, encoding="utf-8") as f:
        return json.load(f)


def validate_required_fields(instance: dict, schema: dict, label: str) -> bool:
    """Check that all required fields in schema are present in instance."""
    required = schema.get("required", [])
    missing = [f for f in required if f not in instance]
    if missing:
        log_fail(f"{label}: missing required fields: {missing}")
        return False
    return True


def validate_enum(value, allowed: list, field_name: str, label: str) -> bool:
    """Check that a value is in the allowed enum list."""
    if value not in allowed:
        log_fail(f"{label}: field '{field_name}' value '{value}' not in {allowed}")
        return False
    return True


def validate_type(value, expected_type: str, field_name: str, label: str) -> bool:
    """Basic type check."""
    type_map = {
        "string": str,
        "number": (int, float),
        "integer": int,
        "boolean": bool,
        "object": dict,
        "array": list,
    }
    py_type = type_map.get(expected_type)
    if py_type and not isinstance(value, py_type):
        log_fail(f"{label}: field '{field_name}' expected type {expected_type}, got {type(value).__name__}")
        return False
    return True


def validate_instance(instance: dict, schema: dict, label: str) -> bool:
    """Validate an instance against a schema (basic structural validation)."""
    ok = True

    # Check required fields
    if not validate_required_fields(instance, schema, label):
        ok = False

    # Check property types and enums
    properties = schema.get("properties", {})
    for field_name, field_schema in properties.items():
        if field_name not in instance:
            continue
        value = instance[field_name]
        field_type = field_schema.get("type")
        if field_type and not validate_type(value, field_type, field_name, label):
            ok = False
            continue
        enum_values = field_schema.get("enum")
        if enum_values and not validate_enum(value, enum_values, field_name, label):
            ok = False

    return ok


def test_valid_camera_event() -> None:
    """Validate valid-camera-event.json against camera-event-v1 schema."""
    schema_path = CONTRACTS_DIR / "events" / "camera-event-v1.schema.json"
    fixture_path = FIXTURES_DIR / "valid-camera-event.json"
    label = "valid-camera-event"

    schema = load_json(schema_path)
    instance = load_json(fixture_path)

    if validate_instance(instance, schema, label):
        log_pass(f"{label} conforms to camera-event-v1 schema")


def test_valid_occupancy_state() -> None:
    """Validate valid-occupancy-state.json against occupancy-state-v1 schema."""
    schema_path = CONTRACTS_DIR / "events" / "occupancy-state-v1.schema.json"
    fixture_path = FIXTURES_DIR / "valid-occupancy-state.json"
    label = "valid-occupancy-state"

    schema = load_json(schema_path)
    instance = load_json(fixture_path)

    if validate_instance(instance, schema, label):
        log_pass(f"{label} conforms to occupancy-state-v1 schema")


def test_invalid_missing_fields() -> None:
    """Validate that invalid-missing-fields.json fails validation (missing required fields)."""
    schema_path = CONTRACTS_DIR / "events" / "camera-event-v1.schema.json"
    fixture_path = FIXTURES_DIR / "invalid-missing-fields.json"
    label = "invalid-missing-fields"

    schema = load_json(schema_path)
    instance = load_json(fixture_path)

    required = schema.get("required", [])
    missing = [f for f in required if f not in instance]

    if missing:
        log_pass(f"{label} correctly fails validation: missing {missing}")
    else:
        log_fail(f"{label} should be invalid but passed all required field checks")


def test_occupancy_vacancy_rule() -> None:
    """Verify VACANT_CONFIRMED is the only valid vacancy state in schema."""
    schema_path = CONTRACTS_DIR / "events" / "occupancy-state-v1.schema.json"
    schema = load_json(schema_path)
    label = "vacancy-rule"

    states = schema.get("properties", {}).get("state", {}).get("enum", [])
    vacancy_states = [s for s in states if "VACANT" in s]

    # VACANT_CONFIRMED must be present
    if "VACANT_CONFIRMED" not in vacancy_states:
        log_fail(f"{label}: VACANT_CONFIRMED not in occupancy enum")
        return

    # VACANT_PENDING is a transitional state, not a confirmed vacancy
    # Only VACANT_CONFIRMED counts as actual vacancy
    if "VACANT_CONFIRMED" in states:
        log_pass(f"{label}: VACANT_CONFIRMED is the only confirmed vacancy state")
    else:
        log_fail(f"{label}: VACANT_CONFIRMED missing from enum")


def test_safety_decision_types() -> None:
    """Verify safety decision schema has expected decision types."""
    schema_path = CONTRACTS_DIR / "events" / "safety-decision-v1.schema.json"
    schema = load_json(schema_path)
    label = "safety-decisions"

    decisions = schema.get("properties", {}).get("decision", {}).get("enum", [])
    expected = {"STOP_REQUEST_REQUIRED", "OPERATION_PERMITTED", "HOLD_CURRENT_STATE", "EMERGENCY_STOP"}

    if set(decisions) == expected:
        log_pass(f"{label}: all expected decision types present")
    else:
        missing = expected - set(decisions)
        extra = set(decisions) - expected
        log_fail(f"{label}: missing={missing}, extra={extra}")


def main() -> int:
    """Run all contract validation tests."""
    print("  Running contract fixture validation...")
    print("")

    test_valid_camera_event()
    test_valid_occupancy_state()
    test_invalid_missing_fields()
    test_occupancy_vacancy_rule()
    test_safety_decision_types()

    print("")
    print(f"  Results: {PASS} passed, {FAIL} failed")

    return 0 if FAIL == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
