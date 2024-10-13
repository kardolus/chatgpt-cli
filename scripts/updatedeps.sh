#!/usr/bin/env bash
set -o pipefail

echo "Checking for unstaged changes..."
if ! git diff --exit-code > /dev/null; then
  echo "Error: You have unstaged changes. Please commit or stash them before running the tests."
  exit 1
fi

go get -u ./...
go mod vendor
go mod tidy

if [[ `git status --porcelain` ]]; then
  echo "Updated dependencies"
  git add .
  git ci -m "Bump dependencies"
  git push
else
  echo "Dependencies up to date"
fi
