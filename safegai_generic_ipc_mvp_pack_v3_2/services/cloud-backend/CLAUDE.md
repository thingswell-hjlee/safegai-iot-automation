# Cloud Backend Instructions

## Scope
TypeScript code for two Lambda handlers: ingest and admin API.

## Boundaries
- No machine control endpoint, MQTT topic, shadow field, or command.
- Validate Gateway identity, schema, size, and idempotency before storage.
- Keep handlers thin; shared domain logic must be testable without AWS.
- Use conditional writes for duplicate events.
- Event images are separate objects, not base64 in metadata JSON.
- Structured logs must avoid credentials, tokens, and personal data.

## Ingest handler
- Event schema validation
- Gateway/certificate/topic identity validation
- Event idempotency
- Gateway last-seen update
- notification policy and cooldown

## Admin API handler
- site/gateway status
- event list/detail
- ACK/resolve/classify
- basic report
- presigned image URL

## Test order
1. Pure domain unit tests
2. AWS client adapter tests with mocks
3. CDK assertions
4. Dev environment smoke
