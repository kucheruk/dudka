#!/usr/bin/env bash
# Task-level contract for P004: root Go module.
# Run: ./scripts/gomod_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "gomod_test FAIL: $*" >&2
  exit 1
}

[[ -f go.mod ]] || fail "go.mod missing"

mod="$(awk '/^module / {print $2; exit}' go.mod)"
[[ "$mod" == "dudka" ]] || fail "module path want dudka, got '${mod:-<empty>}'"

pkgs="$(go list ./...)"
[[ -n "$pkgs" ]] || fail "go list ./... is empty"

echo "$pkgs" | grep -qx 'dudka' \
  || fail "go list ./... must include root package dudka; got: $pkgs"

go test ./... >/dev/null || fail "go test ./... failed"

echo "gomod_test OK"
