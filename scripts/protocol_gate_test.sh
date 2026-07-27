#!/usr/bin/env bash
# Task-level contract for P045: check.sh runs multi-peer protocol tests.
# Run: ./scripts/protocol_gate_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "protocol_gate_test FAIL: $*" >&2
  exit 1
}

[[ -f scripts/check.sh ]] || fail "scripts/check.sh missing"
[[ -x scripts/check.sh ]] || fail "scripts/check.sh must be executable"
[[ -f scripts/protocol_tests.sh ]] || fail "scripts/protocol_tests.sh missing (P045 suite runner)"
[[ -x scripts/protocol_tests.sh ]] || fail "scripts/protocol_tests.sh must be executable"

grep -q 'protocol_tests\.sh' scripts/check.sh \
  || fail "check.sh must invoke scripts/protocol_tests.sh (P045)"

# Suite must include concrete 2+ peer protocol contracts.
for need in announce_test.sh peers_test.sh send_test.sh tail_test.sh wan_test.sh tui_send_test.sh; do
  grep -q "$need" scripts/protocol_tests.sh \
    || fail "protocol_tests.sh must include $need"
done

grep -qiE 'protocol test|protocol_tests' README.md \
  || fail "README.md must document that check.sh runs protocol tests"

out="$(mktemp)"
trap 'rm -f "$out"' EXIT

set +e
./scripts/check.sh >"$out" 2>&1
rc=$?
set -e

[[ "$rc" -eq 0 ]] || fail "check.sh exited $rc; output: $(tail -n 80 "$out")"

grep -q 'protocol_tests: OK' "$out" \
  || fail "check.sh output must include protocol_tests: OK; got: $(tail -n 40 "$out")"
grep -q 'peers_test OK' "$out" \
  || fail "check.sh must run peers_test (2-peer); missing peers_test OK"
grep -q 'send_test OK' "$out" \
  || fail "check.sh must run send_test (2-peer); missing send_test OK"

echo "protocol_gate_test OK"
