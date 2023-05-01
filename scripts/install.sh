#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
mkdir -p bin

TARGET_OS=${1:-darwin}
GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAGS=$(git rev-list --tags --max-count=1)

echo "Target OS: $TARGET_OS"
for b in $(ls cmd); do
  echo -n "Building $b..."

  if [ ! -z "$GIT_TAGS" ]
  then
    GIT_VERSION=$(git describe --tags $GIT_TAGS)
    GOOS=$TARGET_OS go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$GIT_COMMIT -X main.GitVersion=$GIT_VERSION" -o bin/$b -a cmd/$b/main.go
  else
    GOOS=$TARGET_OS go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$GIT_COMMIT" -o bin/$b -a cmd/$b/main.go
  fi

  echo "done"
done
