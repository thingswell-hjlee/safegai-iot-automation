#!/usr/bin/env bash
set -euo pipefail

required=(git make python3 jq)
recommended=(gh go node npm aws sqlite3 shellcheck claude)
missing=0

echo 'Required tools:'
for cmd in "${required[@]}"; do
  if command -v "$cmd" >/dev/null 2>&1; then
    printf '  OK   %-12s %s\n' "$cmd" "$(command -v "$cmd")"
  else
    printf '  MISS %-12s\n' "$cmd"
    missing=1
  fi
done

echo 'Recommended tools:'
for cmd in "${recommended[@]}"; do
  if command -v "$cmd" >/dev/null 2>&1; then
    printf '  OK   %-12s' "$cmd"
    case "$cmd" in
      git) git --version | head -n 1 ;;
      gh) gh --version | head -n 1 ;;
      go) go version ;;
      node) node --version ;;
      npm) npm --version ;;
      aws) aws --version 2>&1 | head -n 1 ;;
      sqlite3) sqlite3 --version | head -n 1 ;;
      shellcheck) shellcheck --version | grep '^version:' | head -n 1 ;;
      claude) claude --version 2>&1 | head -n 1 ;;
      *) echo ;;
    esac
  else
    printf '  WARN %-12s not installed yet\n' "$cmd"
  fi
done

if [ "$missing" -ne 0 ]; then
  echo 'Install missing required tools before development.'
  exit 1
fi
