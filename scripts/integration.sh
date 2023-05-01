#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

if [[ ! -d integration ]]; then
    echo -e "\n\033[0;31m** WARNING  No Integration tests **\033[0m"
    exit 0
fi

echo "Run Integration Tests"
set +e
go test -parallel 1 -timeout 0 -mod=vendor ./integration/... -v -run Integration
exit_code=$?

if [ "$exit_code" != "0" ]; then
    echo -e "\n\033[0;31m** GO Test Failed **\033[0m"
else
    echo -e "\n\033[0;32m** GO Test Succeeded **\033[0m"
fi

exit $exit_code
