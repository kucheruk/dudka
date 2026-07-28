#!/usr/bin/env bash
# Task-level contract for P040/P148: self + remote peers; alone keeps seek.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tui_peers_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true; [[ -n "${pid_b:-}" ]] && kill "$pid_b" 2>/dev/null || true' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

log="$tmpdir/a.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Аня" -listen "127.0.0.1:0" \
  -announce-port 0 -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "dudkad not ready: $(cat "$log")"

frame="$("$tmpdir/dudka" -engine "$listen")"
printf '%s\n' "$frame" | head -n 1 | grep -q '^dudka ' || fail "missing version line: $frame"
printf '%s\n' "$frame" | grep -q 'ДУДКА' || fail "missing brand: $frame"
printf '%s\n' "$frame" | grep -q 'Аня' || fail "missing me: $frame"
printf '%s\n' "$frame" | grep -q 'онлайн 1' || fail "missing self in online count: $frame"
printf '%s\n' "$frame" | grep -q 'Аня · ВЫ' || fail "missing self peer row: $frame"
printf '%s\n' "$frame" | grep -q 'НИКОГО РЯДОМ' || fail "missing empty copy: $frame"

# Second peer → TUI lists them.
port_a="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
port_b="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
# Restart pair on shared announce port for discovery.
kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

log_a="$tmpdir/a2.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a2" -name "Аня" -listen "127.0.0.1:0" \
  -announce-port "$port_a" -announce-target "127.0.0.1:${port_b}" \
  -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "Боря" -listen "127.0.0.1:0" \
  -announce-port "$port_b" -announce-target "127.0.0.1:${port_a}" \
  -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" ]] || fail "pair not ready"

found=0
for _ in $(seq 1 60); do
  frame="$("$tmpdir/dudka" -engine "$listen_a")"
  if printf '%s\n' "$frame" | grep -q 'Боря' && printf '%s\n' "$frame" | grep -q 'онлайн 2' && ! printf '%s\n' "$frame" | grep -q 'НИКОГО РЯДОМ'; then
    found=1
    break
  fi
  sleep 0.1
done
[[ "$found" -eq 1 ]] || fail "TUI missing peer Боря: $($tmpdir/dudka -engine "$listen_a")"

kill "$pid" "$pid_b" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid=""; pid_b=""

echo "tui_peers_test OK"
