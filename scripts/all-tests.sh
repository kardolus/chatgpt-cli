#!/usr/bin/env bash
set -euo pipefail

# Ensure dependencies are tidy and up to date
echo "Tidying go modules and checking for changes..."
go mod tidy
git diff --exit-code go.mod go.sum || {
  echo "go.mod or go.sum has uncommitted changes after running 'go mod tidy'."
  exit 1
}

# Run golangci-lint to check for code issues
echo "Running golangci-lint..."
golangci-lint run

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
./scripts/unit.sh
echo
./scripts/integration.sh
echo
./scripts/contract.sh

