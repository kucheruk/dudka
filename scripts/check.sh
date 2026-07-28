#!/usr/bin/env bash
# Единый локальный/CI гейт: тот же скрипт гоняет агент локально и Forgejo CI.
# После go.mod: go test ./... + multi-peer protocol suite (P045).
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ ! -f go.mod ]]; then
  echo "check.sh: no go.mod yet — nothing to test (ok)"
  exit 0
fi

go test ./...
./scripts/release_contract_test.sh
./scripts/app_icon_test.sh
./scripts/build_windows_app_test.sh
./scripts/build_linux_app_test.sh
./scripts/protocol_tests.sh
