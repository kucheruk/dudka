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

run_until_ready() {
  local args_data="$1" args_name="$2" log="$3" bin="$4"
  local pid
  if [[ -n "$args_name" ]]; then
    "$bin" -data-dir "$args_data" -name "$args_name" -listen "127.0.0.1:0" >"$log" 2>&1 &
  else
    "$bin" -data-dir "$args_data" -listen "127.0.0.1:0" >"$log" 2>&1 &
  fi
  pid=$!
  for _ in $(seq 1 50); do
    if grep -q '^ready ' "$log" 2>/dev/null; then
      kill "$pid" 2>/dev/null || true
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    if ! kill -0 "$pid" 2>/dev/null; then
      fail "dudkad exited early; log: $(cat "$log")"
    fi
    sleep 0.1
  done
  kill "$pid" 2>/dev/null || true
  wait "$pid" 2>/dev/null || true
  fail "timeout waiting for ready; log: $(cat "$log")"
}

go test ./internal/identity/ -count=1 >/dev/null || fail "identity unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

d1="$tmpdir/d1"
log1="$tmpdir/d1a.log"
log1b="$tmpdir/d1b.log"
run_until_ready "$d1" "ТестНик" "$log1" "$bin"
name1="$(sed -n 's/^display_name=//p' "$log1" | head -1)"
[[ "$name1" == "ТестНик" ]] || fail "flag: got $name1"
run_until_ready "$d1" "" "$log1b" "$bin"
name1b="$(sed -n 's/^display_name=//p' "$log1b" | head -1)"
[[ "$name1b" == "ТестНик" ]] || fail "persisted name mismatch: $name1b"

d2="$tmpdir/d2"
log2="$tmpdir/d2a.log"
log2b="$tmpdir/d2b.log"
run_until_ready "$d2" "" "$log2" "$bin"
name2="$(sed -n 's/^display_name=//p' "$log2" | head -1)"
[[ -n "$name2" ]] || fail "empty display_name without -name"
run_until_ready "$d2" "" "$log2b" "$bin"
name2b="$(sed -n 's/^display_name=//p' "$log2b" | head -1)"
[[ "$name2b" == "$name2" ]] || fail "restart display_name changed: $name2 -> $name2b"

echo "displayname_test OK"
