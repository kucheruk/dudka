#!/usr/bin/env bash
# Task-level contract for P010: stable peer_id on disk across restarts.
# Run: ./scripts/peerid_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "peerid_test FAIL: $*" >&2
  exit 1
}

# Run dudkad until ready line, capture stdout, then stop.
run_once() {
  local data="$1" log="$2" bin="$3"
  local pid
  "$bin" -data-dir "$data" -name "PeerIDTest" -listen "127.0.0.1:0" >"$log" 2>&1 &
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

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
data="$tmpdir/data"
bin="$tmpdir/dudkad"
log1="$tmpdir/1.log"
log2="$tmpdir/2.log"

go build -o "$bin" ./cmd/dudkad || fail "go build ./cmd/dudkad failed"

run_once "$data" "$log1" "$bin"
run_once "$data" "$log2" "$bin"

id1="$(sed -n 's/^peer_id=//p' "$log1" | head -1)"
id2="$(sed -n 's/^peer_id=//p' "$log2" | head -1)"

[[ -n "$id1" ]] || fail "first run missing peer_id= line; got: $(cat "$log1")"
[[ "$id1" == "$id2" ]] || fail "restart id mismatch: first=$id1 second=$id2"

[[ -f "$data/peer_id" ]] || fail "peer_id file not on disk"
disk="$(tr -d '[:space:]' <"$data/peer_id")"
[[ "$disk" == "$id1" ]] || fail "disk peer_id=$disk want $id1"

go test ./internal/identity/ >/dev/null || fail "identity unit tests failed"

echo "peerid_test OK"
