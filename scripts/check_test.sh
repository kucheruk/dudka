#!/usr/bin/env bash
# Task-level contract for P003: local/CI gate harness.
# Run: ./scripts/check_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "check_test FAIL: $*" >&2
  exit 1
}

[[ -f scripts/check.sh ]] || fail "scripts/check.sh missing"
[[ -x scripts/check.sh ]] || fail "scripts/check.sh must be executable"

out="$(mktemp)"
trap 'rm -f "$out"' EXIT

set +e
./scripts/check.sh >"$out" 2>&1
rc=$?
set -e

[[ "$rc" -eq 0 ]] || fail "check.sh exited $rc; output: $(cat "$out")"

if [[ ! -f go.mod ]]; then
  grep -qiE 'no go\.mod|nothing to test|ok' "$out" \
    || fail "without go.mod, check.sh should explain no-op success; got: $(cat "$out")"
else
  # With a module, gate must invoke go test (smoke: script source mentions it).
  grep -q 'go test' scripts/check.sh \
    || fail "with go.mod present, check.sh must run go test"
fi

grep -q 'scripts/check.sh' README.md \
  || fail "README.md must document ./scripts/check.sh as the local gate"

echo "check_test OK"
