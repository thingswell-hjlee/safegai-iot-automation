# Gateway Server Instructions

## Scope
Go modular monolith for linux/amd64. It receives camera and I/O events, maintains fail-safe state, executes local warning and stop-request actions, stores evidence, serves local APIs, and queues cloud messages.

## Package direction

```text
cmd -> application -> domain
aadapters -> application/domain
storage -> application/domain
httpapi -> application
domain must not import adapters, cloud, frontend, or vendor packages
```

## Required behavior
- Missing, malformed, delayed, or offline camera data never becomes vacancy.
- Only VACANT_CONFIRMED satisfies vacancy.
- Safety decisions are deterministic and unit tested with truth tables.
- All output commands have command ID, correlation ID, timeout, result, and audit.
- Restart does not replay past pulse commands.
- AWS client is asynchronous through the outbox.
- Vendor-specific payloads remain inside camera adapters.
- No video inference, transcoding, or continuous recording.

## Test order
1. Domain unit tests
2. State transition tests
3. Simulator integration
4. SQLite crash/restart
5. HIL only after T2 approval

## Commands
Run from repository root:
- `make verify-fast`
- `make verify`
- `make build`
