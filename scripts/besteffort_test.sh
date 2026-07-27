#!/usr/bin/env bash
# Task-level contract for P035: no false "delivered" — only accepted/queued.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "besteffort_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/loopback/ -run 'BestEffort|SendStatus|SendQueued|SendLogs|SendResponseBest|SendAndMessages' -count=1 \
  || fail "unit tests failed"

# Production .go (not *_test.go) must not emit delivery-claim log/API tokens.
hits="$(rg -n --glob '*.go' --glob '!*_test.go' \
  -e 'chat_deliver' \
  -e 'StatusDelivered' \
  -e '"delivered"' \
  internal/chat internal/loopback cmd 2>/dev/null || true)"
# Allow the single documenting comment in besteffort.go.
hits="$(printf '%s\n' "$hits" | grep -v 'besteffort.go:.*Never "delivered"' || true)"
if [[ -n "${hits// }" ]]; then
  echo "$hits" >&2
  fail "found delivery-claim tokens in engine code"
fi

echo "besteffort_test OK"
