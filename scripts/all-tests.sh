#!/usr/bin/env bash
set -euo pipefail

log() {
  echo "[$(date +'%Y-%m-%dT%H:%M:%S%z')] $*"
}

# Ensure dependencies are tidy and up to date
log "Tidying Go modules and checking for changes..."
go mod tidy
if ! git diff --exit-code go.mod go.sum; then
  log "go.mod or go.sum has uncommitted changes after 'go mod tidy'."
  exit 1
fi

# Ensure Go code is formatted properly
log "Checking code format with 'go fmt'..."
fmt_output=$(go fmt ./...)
if [ -n "$fmt_output" ]; then
  log "The following files are not formatted properly:"
  echo "$fmt_output"
  log "Please run 'go fmt' to fix the formatting issues."
  exit 1
fi

# Run golangci-lint to check for code issues
log "Running golangci-lint..."
if ! golangci-lint run; then
  log "Linting issues detected."
  exit 1
fi

# Run tests in parallel for faster execution
log "Running unit tests..."
cd "$( dirname "${BASH_SOURCE[0]}" )/.."
./scripts/unit.sh &
unit_pid=$!

log "Running integration tests..."
./scripts/integration.sh &
integration_pid=$!

log "Running contract tests..."
./scripts/contract.sh &
contract_pid=$!

wait $unit_pid $integration_pid $contract_pid
log "All tests completed successfully."
