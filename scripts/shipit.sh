#!/usr/bin/env bash
set -euo pipefail

# Navigate to the project root directory
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Step 1: Check for unstaged changes
echo "Checking for unstaged changes..."
if ! git diff --exit-code > /dev/null; then
  echo "Error: You have unstaged changes. Please commit or stash them before running the tests."
  exit 1
fi

# Step 2: Update dependencies
echo "Updating dependencies..."
./scripts/updatedeps.sh

# Step 3: Run all tests (includes linter, 'go fmt', and 'go mod tidy')
echo "Running all tests..."
./scripts/all-tests.sh

# Step 4: Create binaries
echo "Creating binaries..."
./scripts/binaries.sh

# Step 5: Generate release notes by diffing from the latest tag to HEAD
echo "Generating release notes..."
git diff $(git rev-list --tags --max-count=1)..HEAD | chatgpt -n -p ../prompts/write_release_notes.md for the 'how to update' section explain you can use brew upgrade chatgpt-cli or do a direct download of the binaries for your specific OS. 
