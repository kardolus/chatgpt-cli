#!/usr/bin/env bash
set -euo pipefail

# Navigate to the project root directory
cd "$(dirname "${BASH_SOURCE[0]}")/.."

# Step 1: Check for tag and commit message arguments
if [[ $# -lt 2 ]]; then
  echo "Error: Missing arguments. Usage: ./shipit.sh <tag> <message>"
  exit 1
fi

TAG="$1"
MESSAGE="$2"

# Step 2: Check for unstaged changes
echo "Checking for unstaged changes..."
if ! git diff --exit-code > /dev/null; then
  echo "Error: You have unstaged changes. Please commit or stash them before running the tests."
  exit 1
fi

# Step 3: Update dependencies
echo "Updating dependencies..."
./scripts/updatedeps.sh

# Step 4: Run all tests (includes linter, 'go fmt', and 'go mod tidy')
echo "Running all tests..."
./scripts/all-tests.sh

# Step 5: Create and push git tag
echo "Creating git tag..."
git tag -a "$TAG" -m "$MESSAGE"
git push origin --tags

# Step 6: Create binaries
echo "Creating binaries..."
./scripts/binaries.sh

# Step 7: Generate release notes by diffing from the latest tag to HEAD
echo "Generating release notes..."
git diff "$(git rev-list --tags --max-count=1)"..HEAD  -- . ":(exclude)vendor" | chatgpt -n -p ../prompts/write_release_notes.md for the 'how to update' section explain you can use brew upgrade chatgpt-cli or do a direct download of the binaries for your specific OS. The version we are releasing is "$TAG"

echo "Release complete. Tag $TAG has been created, pushed, and binaries are ready."