#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

# Clean up any previous coverage data
rm -f coverage.out coverage_unit.out coverage_integration.out coverage_contract.out

# Run Unit Tests with coverage
echo "Run Unit Tests with Coverage"
CONFIG_PATH="file://$PWD" TESTING=true go test -mod=vendor ./... -v -cover -coverpkg=./... -coverprofile=coverage_unit.out -run Unit

# Run Integration Tests with coverage
echo "Run Integration Tests with Coverage"
CONFIG_PATH="file://$PWD" TESTING=true go test -mod=vendor ./test/... -v -cover -coverpkg=./... -coverprofile=coverage_integration.out -run Integration

# Run Contract Tests with coverage
echo "Run Contract Tests with Coverage"
CONFIG_PATH="file://$PWD" TESTING=true go test -mod=vendor ./test/... -v -cover -coverpkg=./... -coverprofile=coverage_contract.out -run Contract

# Merge the coverage profiles
echo "Merging Coverage Profiles"
echo "mode: set" > coverage.out
tail -q -n +2 coverage_unit.out coverage_integration.out coverage_contract.out >> coverage.out || true

# Generate an HTML report
echo "Generating HTML Coverage Report"
go tool cover -html=coverage.out -o test/coverage.html

echo -e "\n\033[0;32m** Coverage Report Generated: test/coverage.html **\033[0m"

rm -f coverage.out coverage_unit.out coverage_integration.out coverage_contract.out
