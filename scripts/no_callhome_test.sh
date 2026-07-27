#!/usr/bin/env bash
# P096: no callhome / update-check / WAN telemetry clients in source or release binary.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() { echo "no_callhome_test FAIL: $*" >&2; exit 1; }

# Source: forbid obvious callhome patterns outside docs/tests.
if rg -n --glob '!docs/**' --glob '!*_test.go' --glob '!**/papercuts/**' \
  -e 'callhome' -e 'update[_-]?check' -e 'telemetry\.(zamoo|dudka)' \
  -e 'https?://[^"]*update' \
  --type-add 'go:*.go' -t go -t dart . 2>/dev/null | rg -v 'no_callhome|P096|без callhome|callhome не'; then
  fail "forbidden callhome-ish strings in source"
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
CGO_ENABLED=0 go build -o "$tmpdir/dudkad" ./cmd/dudkad
CGO_ENABLED=0 go build -o "$tmpdir/dudka" ./cmd/dudka

for bin in dudkad dudka; do
  if strings "$tmpdir/$bin" | rg -i -e 'callhome' -e 'update-check' -e 'update_check' -e 'telemetry\.zamoo' >/dev/null; then
    fail "binary $bin contains callhome/update strings"
  fi
done

echo "no_callhome_test OK"
