#!/usr/bin/env bash
# Task-level contract for P042: Enter/send — two peers exchange text via TUI client.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tui_send_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -run 'ClientSend|HandleCompose|RenderShowsInput' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Аня" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "Боря" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""; listen_b=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "engines not ready"

# Wait mutual discovery.
for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then break; fi
  sleep 0.1
done

# A sends via TUI -send; B's TUI frame must show the text.
frame_a="$("$tmpdir/dudka" -engine "$listen_a" -send "привет от Ани")"
printf '%s\n' "$frame_a" | grep -q 'привет от Ани' || fail "A frame missing own send:\n$frame_a"

seen_b=0
for _ in $(seq 1 40); do
  frame_b="$("$tmpdir/dudka" -engine "$listen_b")"
  if printf '%s\n' "$frame_b" | grep -q 'привет от Ани'; then
    seen_b=1
    break
  fi
  sleep 0.1
done
[[ "$seen_b" -eq 1 ]] || fail "B missing A's text: $($tmpdir/dudka -engine "$listen_b")"

# B replies via TUI; A sees it.
"$tmpdir/dudka" -engine "$listen_b" -send "ответ Бори" >/dev/null
seen_a=0
for _ in $(seq 1 40); do
  frame_a="$("$tmpdir/dudka" -engine "$listen_a")"
  if printf '%s\n' "$frame_a" | grep -q 'ответ Бори'; then
    seen_a=1
    break
  fi
  sleep 0.1
done
[[ "$seen_a" -eq 1 ]] || fail "A missing B's text: $($tmpdir/dudka -engine "$listen_a")"

# stdin Enter path: pipe one line into -watch (HandleComposeLine).
printf 'через stdin\n' | "$tmpdir/dudka" -engine "$listen_a" -watch -interval 10s >/dev/null
seen=0
for _ in $(seq 1 40); do
  if "$tmpdir/dudka" -engine "$listen_b" | grep -q 'через stdin'; then
    seen=1
    break
  fi
  sleep 0.1
done
[[ "$seen" -eq 1 ]] || fail "B missing stdin-composed text"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "tui_send_test OK"
