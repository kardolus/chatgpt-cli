#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
mkdir -p bin

GIT_COMMIT=$(git rev-list -1 HEAD)
GIT_TAGS=$(git rev-list --tags --max-count=1)

# Add an array of common OSes and architectures
TARGETS=(
  "darwin:amd64"
  "darwin:arm64"
  "linux:amd64"
  "linux:arm64"
  "linux:386"
  "windows:amd64"
  "freebsd:amd64"
  "freebsd:arm64"
)

for b in $(ls cmd); do
  for target in "${TARGETS[@]}"; do
    IFS=":" read -ra os_arch <<< "$target"
    os="${os_arch[0]}"
    arch="${os_arch[1]}"

    binary_name="$b-$os-$arch"
    [ "$os" == "windows" ] && binary_name="$b-$os-$arch.exe"

    echo -n "Building $b for $os/$arch..."

    if [ ! -z "$GIT_TAGS" ]; then
      GIT_VERSION=$(git describe --tags $GIT_TAGS)
      GOOS=$os GOARCH=$arch go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$GIT_COMMIT -X main.GitVersion=$GIT_VERSION" -o "bin/$binary_name" -a cmd/$b/main.go
    else
      GOOS=$os GOARCH=$arch go build -mod=vendor -ldflags="-s -w -X main.GitCommit=$GIT_COMMIT" -o "bin/$binary_name" -a cmd/$b/main.go
    fi

    echo "done"
  done
done
