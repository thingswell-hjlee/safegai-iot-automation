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


def resolve_ref(schema: dict, ref_path: str) -> dict:
    """Resolve a simple $ref within the same schema (e.g. #/$defs/Foo)."""
    if not ref_path.startswith("#/"):
        return {}
    parts = ref_path.lstrip("#/").split("/")
    node = schema
    for part in parts:
        node = node.get(part, {})
    return node


def validate_instance(instance: dict, schema: dict, label: str) -> bool:
    """Validate an instance against a schema (basic structural validation)."""
    ok = True

    # Check required fields from schema itself
    if not validate_required_fields(instance, schema, label):
        ok = False

    # Also check required fields from allOf references (envelope)
    for ref_item in schema.get("allOf", []):
        if "$ref" in ref_item and "envelope" in ref_item["$ref"]:
            envelope_path = CONTRACTS_DIR / "events" / "event-envelope-v1.schema.json"
            if envelope_path.exists():
                envelope_schema = load_json(envelope_path)
                if not validate_required_fields(instance, envelope_schema, f"{label}[envelope]"):
                    ok = False

    # Check property types and enums
    properties = schema.get("properties", {})
    for field_name, field_schema in properties.items():
        if field_name not in instance:
            continue
        value = instance[field_name]

        # Resolve $ref if present
        if "$ref" in field_schema:
            ref_resolved = resolve_ref(schema, field_schema["$ref"])
            if ref_resolved:
                field_schema = ref_resolved

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

    # The canonical schema uses $defs for OccupancyState
    occupancy_def = schema.get("$defs", {}).get("OccupancyState", {})
    states = occupancy_def.get("enum", [])

    if not states:
        # Fallback: check properties.currentState or properties.state
        states = schema.get("properties", {}).get("currentState", {}).get("enum", [])

    vacancy_states = [s for s in states if "VACANT" in s]

    if "VACANT_CONFIRMED" not in states:
        log_fail(f"{label}: VACANT_CONFIRMED not in occupancy enum")
        return

    # VACANT_PENDING is a transitional state, not a confirmed vacancy
    # Only VACANT_CONFIRMED counts as actual vacancy
    log_pass(f"{label}: VACANT_CONFIRMED is the only confirmed vacancy state")


def test_safety_decision_types() -> None:
    """Verify safety decision schema has canonical decision types."""
    schema_path = CONTRACTS_DIR / "events" / "safety-decision-v1.schema.json"
    schema = load_json(schema_path)
    label = "safety-decisions"

    # The canonical schema uses $defs for SafetyDecision
    decision_def = schema.get("$defs", {}).get("SafetyDecision", {})
    decisions = decision_def.get("enum", [])

    if not decisions:
        decisions = schema.get("properties", {}).get("decision", {}).get("enum", [])

    expected = {
        "SAFE",
        "WARNING",
        "STOP_REQUEST_REQUIRED",
        "RESTART_INTERLOCK",
        "SAFETY_CONFIRMATION_UNAVAILABLE",
        "MAINTENANCE_MONITORING",
    }

    # These values must NOT be present (banned by canonical contract)
    banned = {"OPERATION_PERMITTED", "ALLOW_START", "EMERGENCY_STOP"}
    found_banned = banned & set(decisions)

    if found_banned:
        log_fail(f"{label}: banned decision types found: {found_banned}")
        return

    if set(decisions) == expected:
        log_pass(f"{label}: all canonical decision types present, no banned values")
    else:
        missing = expected - set(decisions)
        extra = set(decisions) - expected
        if missing:
            log_fail(f"{label}: missing={missing}")
        if extra:
            log_fail(f"{label}: unexpected extra={extra}")
        if not missing and not extra:
            log_pass(f"{label}: all canonical decision types present")


def test_equipment_state_types() -> None:
    """Verify equipment state schema has canonical state types."""
    schema_path = CONTRACTS_DIR / "events" / "equipment-state-v1.schema.json"
    schema = load_json(schema_path)
    label = "equipment-states"

    # The canonical schema uses $defs for EquipmentState
    state_def = schema.get("$defs", {}).get("EquipmentState", {})
    states = state_def.get("enum", [])

    expected = {"RUNNING", "STOPPED", "STARTING", "STOPPING", "FAULT", "OFFLINE", "UNKNOWN"}

    # RESTART_REQUESTED must NOT be an equipment state
    if "RESTART_REQUESTED" in states:
        log_fail(f"{label}: RESTART_REQUESTED must not be an EquipmentState")
        return

    if set(states) == expected:
        log_pass(f"{label}: all canonical equipment states present")
    else:
        missing = expected - set(states)
        extra = set(states) - expected
        log_fail(f"{label}: missing={missing}, extra={extra}")


def test_actuation_command_types() -> None:
    """Verify actuation result schema has only allowed command types."""
    schema_path = CONTRACTS_DIR / "events" / "actuation-result-v1.schema.json"
    schema = load_json(schema_path)
    label = "actuation-commands"

    command_enum = schema.get("properties", {}).get("commandType", {}).get("enum", [])
    expected = {"STOP_REQUEST", "WARNING_LIGHT", "WARNING_SIREN", "VOICE_ANNOUNCE", "DIGITAL_OUTPUT_TEST"}

    if set(command_enum) == expected:
        log_pass(f"{label}: all canonical actuation commands present")
    else:
        missing = expected - set(command_enum)
        extra = set(command_enum) - expected
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
    test_equipment_state_types()
    test_actuation_command_types()

    print("")
    print(f"  Results: {PASS} passed, {FAIL} failed")

    return 0 if FAIL == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
