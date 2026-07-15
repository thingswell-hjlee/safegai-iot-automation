SHELL := /usr/bin/env bash
.DEFAULT_GOAL := help

VERSION ?= 0.1.0
ARCH ?= amd64

.PHONY: help setup check-prereqs format lint typecheck test test-contract test-integration security build package-amd64 verify-fast verify check-json check-shell manifest

help:
	@printf '%s\n' \
	  'SafeGAI development commands' \
	  '  make check-prereqs   Check required tools' \
	  '  make format          Format files when components exist' \
	  '  make lint            Run linters when components exist' \
	  '  make typecheck       Run type checks when components exist' \
	  '  make test            Run unit tests when components exist' \
	  '  make test-contract   Validate contract files' \
	  '  make build           Build components when components exist' \
	  '  make package-amd64   Build Debian package' \
	  '  make manifest        Generate release manifest' \
	  '  make verify-fast     Fast local verification' \
	  '  make verify          Full pre-PR verification'

setup: check-prereqs
	@./scripts/bootstrap-repo.sh

check-prereqs:
	@./scripts/check-prereqs.sh

format:
	@if find services/gateway-server -type f -name '*.go' -print -quit 2>/dev/null | grep -q .; then \
	  gofmt -w $$(find services/gateway-server -type f -name '*.go'); \
	fi
	@if [ -f package.json ]; then npm run format --if-present; fi

lint:
	@if [ -f services/gateway-server/go.mod ]; then \
	  (cd services/gateway-server && go vet ./...); \
	fi
	@if [ -f package.json ]; then npm run lint --if-present; fi

check-json:
	@./scripts/check-json.py

check-shell:
	@if command -v shellcheck >/dev/null 2>&1; then \
	  files=$$(find scripts .claude/hooks infra/edge/scripts -type f -name '*.sh' 2>/dev/null); \
	  if [ -n "$$files" ]; then shellcheck $$files; fi; \
	else \
	  echo 'shellcheck not installed; skipping'; \
	fi

typecheck:
	@if [ -f package.json ]; then npm run typecheck --if-present; fi

test:
	@if [ -f services/gateway-server/go.mod ]; then \
	  (cd services/gateway-server && go test ./...); \
	fi
	@if [ -f package.json ]; then npm test --if-present; fi

test-contract: check-json
	@./scripts/run-contract-tests.sh

test-integration:
	@if [ -f package.json ]; then npm run test:integration --if-present; fi

security:
	@if command -v git >/dev/null 2>&1; then \
	  if git grep -nE 'BEGIN (RSA|EC|OPENSSH|PRIVATE) PRIVATE KEY' -- . ':!*.example' >/tmp/safegai-secret-scan.txt 2>/dev/null; then \
	    cat /tmp/safegai-secret-scan.txt; \
	    echo 'Potential private key material found'; \
	    exit 1; \
	  fi; \
	fi

build:
	@if [ -f services/gateway-server/go.mod ]; then \
	  mkdir -p dist; \
	  (cd services/gateway-server && CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o ../../dist/safegai-edge ./cmd/safegai-edge); \
	fi
	@if [ -f package.json ]; then npm run build --if-present; fi

package-amd64: build
	@VERSION=$(VERSION) packaging/gateway/build-deb.sh

manifest: build
	@VERSION=$(VERSION) scripts/generate-manifest.sh

verify-fast: format check-json check-shell lint typecheck test test-contract security
	@echo 'verify-fast passed'

verify: verify-fast build test-integration
	@echo 'verify passed'
