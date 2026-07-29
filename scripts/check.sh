#!/usr/bin/env bash
# Единый локальный/CI гейт: тот же скрипт гоняет агент локально и Forgejo CI.
# После go.mod: unit/integration tests, release and packaging contracts.
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ ! -f go.mod ]]; then
  echo "check.sh: no go.mod yet — nothing to test (ok)"
  exit 0
fi

go test ./...
./scripts/release_contract_test.sh
./scripts/app_icon_test.sh
./scripts/web_asset_contract_test.sh
./scripts/build_windows_app_test.sh
./scripts/build_linux_tui_test.sh
