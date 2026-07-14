#!/usr/bin/env python3
import json
from pathlib import Path

bad: list[str] = []
for path in Path('.').rglob('*.json'):
    if any(part in {'.git', 'node_modules', 'dist', 'build'} for part in path.parts):
        continue
    try:
        json.loads(path.read_text(encoding='utf-8'))
    except Exception as exc:  # noqa: BLE001
        bad.append(f'{path}: {exc}')

if bad:
    raise SystemExit('\n'.join(bad))

print('JSON syntax OK')
