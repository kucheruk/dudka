#!/usr/bin/env bash
# Task-level contract for P010: stable peer_id on disk across restarts.
# Run: ./scripts/peerid_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "peerid_test FAIL: $*" >&2
  exit 1
}

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
data="$tmpdir/data"
bin="$tmpdir/dudkad"

go build -o "$bin" ./cmd/dudkad || fail "go build ./cmd/dudkad failed"

out1="$("$bin" -data-dir "$data")"
out2="$("$bin" -data-dir "$data")"

id1="$(printf '%s\n' "$out1" | sed -n 's/^peer_id=//p' | head -1)"
id2="$(printf '%s\n' "$out2" | sed -n 's/^peer_id=//p' | head -1)"

[[ -n "$id1" ]] || fail "first run missing peer_id= line; got: $out1"
[[ "$id1" == "$id2" ]] || fail "restart id mismatch: first=$id1 second=$id2"

[[ -f "$data/peer_id" ]] || fail "peer_id file not on disk"
disk="$(tr -d '[:space:]' <"$data/peer_id")"
[[ "$disk" == "$id1" ]] || fail "disk peer_id=$disk want $id1"

go test ./internal/identity/ >/dev/null || fail "identity unit tests failed"

echo "peerid_test OK"
