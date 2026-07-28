#!/usr/bin/env bash
# Task-level contract for P033: third peer GET /tail matches keeper after register.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tail_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/discovery/ ./internal/loopback/ -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-} ${pid_c:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

# Fixed peer_ids so "peer-a" is lexicographic keeper among a/b/c.
mkdir -p "$tmpdir/a" "$tmpdir/b" "$tmpdir/c"
printf 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa\n' >"$tmpdir/a/peer_id"
printf 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb\n' >"$tmpdir/b/peer_id"
printf 'cccccccc-cccc-4ccc-8ccc-cccccccccccc\n' >"$tmpdir/c/peer_id"

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
port_c="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"; log_c="$tmpdir/c.log"
"$bin" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port_a" -announce-target "127.0.0.1:${port_b}" \
  -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port_b" -announce-target "127.0.0.1:${port_a}" \
  -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
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
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "A/B not ready"

# Wait mutual peers, then seed chat on keeper (Alice = min peer_id).
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

for text in one two three; do
  curl -sS --max-time 2 -X POST "http://${listen_a}/send" \
    -H 'Content-Type: application/json' \
    -d "{\"text\":\"$text\"}" >/dev/null || fail "send $text"
done

# Join Carol (third peer).
"$bin" -data-dir "$tmpdir/c" -name "Carol" -listen "127.0.0.1:0" \
  -announce-port "$port_c" -announce-target "127.0.0.1:${port_a}" \
  -session-port 0 -announce-interval 150ms >"$log_c" 2>&1 &
pid_c=$!

listen_c=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_c" 2>/dev/null; then
    listen_c="$(grep '^listen=' "$log_c" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_c" ]] || fail "C not ready: $(cat "$log_c")"

matched=0
for _ in $(seq 1 80); do
  ta="$(curl -sS --max-time 1 "http://${listen_a}/tail" || true)"
  tc="$(curl -sS --max-time 1 "http://${listen_c}/tail" || true)"
  if python3 - "$ta" "$tc" <<'PY'
import json, sys
a, c = json.loads(sys.argv[1]), json.loads(sys.argv[2])
if a.get("keeper_id") != "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa":
    raise SystemExit(1)
if not a.get("is_keeper") or c.get("is_keeper"):
    raise SystemExit(1)
ma = a.get("messages") or []
mc = c.get("messages") or []
if len(ma) < 3 or len(ma) != len(mc):
    raise SystemExit(1)
ids_a = [m.get("msg_id") for m in ma]
ids_c = [m.get("msg_id") for m in mc]
texts_a = [m.get("text") for m in ma]
raise SystemExit(0 if ids_a == ids_c and set(texts_a) >= {"one", "two", "three"} else 1)
PY
  then
    matched=1
    break
  fi
  sleep 0.1
done
[[ "$matched" -eq 1 ]] || fail "C tail != keeper; a=$(curl -sS "http://${listen_a}/tail") c=$(curl -sS "http://${listen_c}/tail")"

for p in "$pid_a" "$pid_b" "$pid_c"; do kill "$p" 2>/dev/null || true; done
for p in "$pid_a" "$pid_b" "$pid_c"; do wait "$p" 2>/dev/null || true; done
pid_a=""; pid_b=""; pid_c=""

echo "tail_test OK"
