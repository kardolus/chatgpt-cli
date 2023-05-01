#!/usr/bin/env bash
set -o pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

go clean -testcache

echo "Run Unit Tests"
if [ -z "$1" ]
then
    CONFIG_PATH="file://$PWD" TESTING=true go test -mod=vendor ./... -v -run Unit
else
    CONFIG_PATH="file://$PWD" TESTING=true go test -mod=vendor ./"$1" -v -run Unit
fi
exit_code=$?

if [ "$exit_code" != "0" ]; then
    echo -e "\n\033[0;31m** GO Test Failed **\033[0m"
else
    echo -e "\n\033[0;32m** GO Test Succeeded **\033[0m"
fi

exit $exit_code
