#!/usr/bin/env bash
# P096/DUD-UPD-101: no callhome/telemetry; updater has one exact WAN manifest.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() { echo "no_callhome_test FAIL: $*" >&2; exit 1; }

# Source: forbid callhome/telemetry patterns outside docs/tests.
if rg -n --glob '!docs/**' --glob '!*_test.go' --glob '!**/papercuts/**' \
  -e 'callhome' -e 'telemetry\.(zamoo|dudka)' \
  --type-add 'go:*.go' -t go -t dart . 2>/dev/null | rg -v 'no_callhome|P096|без callhome|callhome не'; then
  fail "forbidden callhome/telemetry strings in source"
fi

urls="$(rg -o --no-filename "https://[^\"' ]+" apps/dudka/lib -g '*.dart' | sort -u || true)"
unexpected="$(printf '%s\n' "$urls" | rg -v '^https://zamoo\.team/dudka/update\.json$' || true)"
[[ -z "$unexpected" ]] || fail "unexpected GUI WAN URL(s): $unexpected"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
CGO_ENABLED=0 go build -o "$tmpdir/dudkad" ./cmd/dudkad
CGO_ENABLED=0 go build -o "$tmpdir/dudka" ./cmd/dudka

for bin in dudkad dudka; do
  if strings "$tmpdir/$bin" | rg -i -e 'callhome' -e 'telemetry\.zamoo' >/dev/null; then
    fail "binary $bin contains callhome/telemetry strings"
  fi
done

echo "no_callhome_test OK"
