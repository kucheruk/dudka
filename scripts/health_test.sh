#!/usr/bin/env bash
# Task-level contract for P012: ready line + loopback GET /health → 200.
# Run: ./scripts/health_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "health_test FAIL: $*" >&2
  exit 1
}

go test ./internal/loopback/ -count=1 >/dev/null || fail "loopback unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
bin="$tmpdir/dudkad"
log="$tmpdir/out.log"
data="$tmpdir/data"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

"$bin" -data-dir "$data" -name "HealthNick" -listen "127.0.0.1:0" >"$log" 2>&1 &
pid=$!

ready=""
listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    ready="$(grep '^ready ' "$log" | head -n 1)"
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  if ! kill -0 "$pid" 2>/dev/null; then
    fail "dudkad exited early; log: $(cat "$log")"
  fi
  sleep 0.1
done

[[ -n "$ready" ]] || fail "no ready line; log: $(cat "$log")"
[[ "$ready" == ready\ peer_id=*\ name=HealthNick ]] || fail "ready line mismatch: $ready"
[[ -n "$listen" ]] || fail "no listen= line"

code="$(curl -sS -o "$tmpdir/body" -w '%{http_code}' --max-time 2 "http://${listen}/health")"
[[ "$code" == "200" ]] || fail "GET /health status=$code body=$(cat "$tmpdir/body")"
[[ "$(cat "$tmpdir/body")" == "ok" ]] || fail "health body want ok, got $(cat "$tmpdir/body")"

# Non-goal P015: /me must not exist yet (404).
me_code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 2 "http://${listen}/me" || true)"
[[ "$me_code" == "404" ]] || fail "GET /me should be 404 before P015, got $me_code"

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "health_test OK"
