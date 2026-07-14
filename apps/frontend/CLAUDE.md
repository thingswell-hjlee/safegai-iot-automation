# Frontend Instructions

## Scope
One React/TypeScript application supporting local and cloud data adapters and three role modes.

## Role rules
- USER: status, video, warning, action guide only
- OPERATOR: event ACK, resolve, classify, work window, reports
- MAINTAINER: local-only device, I/O, diagnostics, backup, update workflows

Hiding a control is not authorization. Backend APIs must enforce the same role.

## Safety UX
- UNKNOWN and STALE must never appear as safe or vacant.
- Use color plus icon plus text.
- Active critical warning remains visible.
- Confirmation is required for maintenance and TEST transitions.
- User mode must not expose ACK, safety mapping, or I/O test.

## Performance
- Four H.264 substreams are proxied; do not add browser or gateway transcoding.
- Reconnect streams independently.
- Show offline placeholders without blocking the rest of the page.

## Test order
- component tests
- role authorization tests
- keyboard and large-text tests
- local/cloud adapter contract tests
- browser memory test during four-stream display
