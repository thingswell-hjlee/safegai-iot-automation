# SafeGAI Unit Tests

## Conventions

### Directory Structure

```
tests/unit/
  gateway/        - Go gateway server unit tests (run via go test)
  state-engine/   - Zone state engine logic tests
  adapters/       - Camera and device adapter tests
  decisions/      - Safety decision logic tests
```

### Naming

- Test files follow the pattern `<module>_test.<ext>`
- Go tests: `*_test.go` in the same package
- Python tests: `test_*.py` using standard unittest or pytest conventions

### Running Tests

```bash
# Run all unit tests
make test

# Run Go gateway tests only
cd services/gateway-server && go test ./...

# Run Python tests if present
python3 -m pytest tests/unit/ -v
```

### Test Requirements

1. **No external dependencies**: Tests must use only stdlib or project-internal packages
2. **No network calls**: All external services must be mocked
3. **No real hardware**: Camera, PLC, and sensor interactions use simulators
4. **Deterministic**: Tests must not depend on timing or random values
5. **Isolated**: Each test sets up and tears down its own state
6. **Fast**: Unit tests should complete within seconds

### Safety-Related Tests

Tests that validate safety rules must:
- Explicitly document which safety rule is being tested
- Cover both positive (allowed) and negative (blocked) cases
- Verify that VACANT_CONFIRMED is the only vacancy state treated as safe
- Verify that UNKNOWN and STALE always block equipment operation
- Include boundary conditions (e.g., confidence thresholds)

### Coverage

- Aim for 80% line coverage on safety-critical paths
- Decision logic must have 100% branch coverage
- Document any intentionally uncovered code with rationale
