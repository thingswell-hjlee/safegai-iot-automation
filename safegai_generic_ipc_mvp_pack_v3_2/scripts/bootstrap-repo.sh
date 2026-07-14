#!/usr/bin/env bash
set -euo pipefail

mkdir -p \
  contracts/events contracts/api contracts/mqtt \
  services/gateway-server services/cloud-backend \
  apps/frontend infra/aws \
  simulators/camera simulators/io \
  tests/acceptance tests/hil tests/evidence \
  packaging/gateway docs/evidence dist

for dir in \
  contracts/events contracts/api contracts/mqtt \
  services/gateway-server services/cloud-backend \
  apps/frontend infra/aws \
  simulators/camera simulators/io \
  tests/acceptance tests/hil tests/evidence \
  packaging/gateway; do
  if [ ! -e "$dir/README.md" ]; then
    printf '# %s\n\nImplementation is created through an approved GitHub issue.\n' "$dir" > "$dir/README.md"
  fi
done

echo 'Repository directories are ready.'
