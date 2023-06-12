#!/usr/bin/env bash
set -euo pipefail

if [ $# -eq 0 ]; then
    echo "No arguments provided"
    exit 1
fi

test_type=$1

cd "$( dirname "${BASH_SOURCE[0]}" )/.."

if [[ ! -d integration ]]; then
    echo -e "\n\033[0;31m** WARNING  No ${test_type} tests **\033[0m"
    exit 0
fi

echo "Run ${test_type} Tests"
set +e
go test -parallel 1 -timeout 0 -mod=vendor ./integration/... -v -run ${test_type}
exit_code=$?

if [ "$exit_code" != "0" ]; then
    echo -e "\n\033[0;31m** GO ${test_type} Test Failed **\033[0m"
else
    echo -e "\n\033[0;32m** GO ${test_type} Test Succeeded **\033[0m"
fi

exit $exit_code

