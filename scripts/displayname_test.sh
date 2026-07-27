#!/usr/bin/env bash
# Task-level contract for P011: display_name flag / hostname / generated.
# Run: ./scripts/displayname_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "displayname_test FAIL: $*" >&2
  exit 1
}

go test ./internal/identity/ -count=1 >/dev/null || fail "identity unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

# Flag branch: -name wins and persists.
d1="$tmpdir/d1"
out1="$("$bin" -data-dir "$d1" -name "ТестНик")"
name1="$(printf '%s\n' "$out1" | sed -n 's/^display_name=//p' | head -1)"
[[ "$name1" == "ТестНик" ]] || fail "flag: got $name1"
out1b="$("$bin" -data-dir "$d1")"
name1b="$(printf '%s\n' "$out1b" | sed -n 's/^display_name=//p' | head -1)"
[[ "$name1b" == "ТестНик" ]] || fail "persisted name mismatch: $name1b"

# No flag / no TTY prompt: hostname or generated; must be non-empty and stable.
d2="$tmpdir/d2"
out2="$("$bin" -data-dir "$d2")"
name2="$(printf '%s\n' "$out2" | sed -n 's/^display_name=//p' | head -1)"
[[ -n "$name2" ]] || fail "empty display_name without -name"
out2b="$("$bin" -data-dir "$d2")"
name2b="$(printf '%s\n' "$out2b" | sed -n 's/^display_name=//p' | head -1)"
[[ "$name2b" == "$name2" ]] || fail "restart display_name changed: $name2 -> $name2b"

echo "displayname_test OK"
