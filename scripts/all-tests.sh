#!/usr/bin/env bash
set -euo pipefail

cd "$( dirname "${BASH_SOURCE[0]}" )/.."
./scripts/unit.sh && ./scripts/integration.sh
