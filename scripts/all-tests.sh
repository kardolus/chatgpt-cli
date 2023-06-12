#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
./scripts/unit.sh
echo
./scripts/integration.sh
echo
./scripts/contract.sh

