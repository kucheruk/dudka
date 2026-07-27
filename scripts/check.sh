#!/usr/bin/env bash
# Единый локальный/CI гейт: тот же скрипт гоняет агент локально и Forgejo CI.
# Пока нет go.mod — no-op success (фаза 0). После появления модуля — go test ./...
set -euo pipefail
cd "$(dirname "$0")/.."

if [[ ! -f go.mod ]]; then
  echo "check.sh: no go.mod yet — nothing to test (ok)"
  exit 0
fi

go test ./...
