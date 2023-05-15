#!/usr/bin/env bash
set -o pipefail

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
