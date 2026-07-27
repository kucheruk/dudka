#!/usr/bin/env bash
# Task-level contract for P034: keeper leaves → re-election → third peer gets tail.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "keeper_leave_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/discovery/ ./internal/loopback/ -run 'KeeperLeave|PeerStorePrune|ThirdPeer' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-} ${pid_c:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

mkdir -p "$tmpdir/a" "$tmpdir/b" "$tmpdir/c"
printf 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa\n' >"$tmpdir/a/peer_id"
printf 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb\n' >"$tmpdir/b/peer_id"
printf 'cccccccc-cccc-4ccc-8ccc-cccccccccccc\n' >"$tmpdir/c/peer_id"

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

# Interval 150ms → default PeerTTL 750ms.
log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"; log_c="$tmpdir/c.log"
"$bin" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
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
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "A/B not ready"

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

for text in keep-1 keep-2; do
  curl -sS --max-time 2 -X POST "http://${listen_a}/send" \
    -H 'Content-Type: application/json' \
    -d "{\"text\":\"$text\"}" >/dev/null || fail "send $text"
done

# Wait Bob has messages, then kill keeper Alice.
for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if python3 - "$mb" <<'PY'
import json, sys
msgs = json.loads(sys.argv[1]).get("messages") or []
raise SystemExit(0 if len(msgs) >= 2 else 1)
PY
  then break; fi
  sleep 0.1
done

kill "$pid_a" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
pid_a=""

# Wait until Bob drops Alice and becomes keeper.
reelected=0
for _ in $(seq 1 40); do
  tb="$(curl -sS --max-time 1 "http://${listen_b}/tail" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$tb" "$pb" <<'PY'
import json, sys
tail, peers = json.loads(sys.argv[1]), json.loads(sys.argv[2])
ids = {p.get("peer_id") for p in (peers.get("peers") or [])}
alice = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa"
bob = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
raise SystemExit(0 if (alice not in ids and tail.get("keeper_id") == bob and tail.get("is_keeper")) else 1)
PY
  then
    reelected=1
    break
  fi
  sleep 0.1
done
[[ "$reelected" -eq 1 ]] || fail "B did not become keeper; tail=$(curl -sS "http://${listen_b}/tail") peers=$(curl -sS "http://${listen_b}/peers")"

"$bin" -data-dir "$tmpdir/c" -name "Carol" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_c" 2>&1 &
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
  tb="$(curl -sS --max-time 1 "http://${listen_b}/tail" || true)"
  tc="$(curl -sS --max-time 1 "http://${listen_c}/tail" || true)"
  if python3 - "$tb" "$tc" <<'PY'
import json, sys
b, c = json.loads(sys.argv[1]), json.loads(sys.argv[2])
bob = "bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb"
if b.get("keeper_id") != bob or c.get("keeper_id") != bob:
    raise SystemExit(1)
if not b.get("is_keeper") or c.get("is_keeper"):
    raise SystemExit(1)
mb, mc = b.get("messages") or [], c.get("messages") or []
if len(mb) < 2 or len(mb) != len(mc):
    raise SystemExit(1)
ids_b = [m.get("msg_id") for m in mb]
ids_c = [m.get("msg_id") for m in mc]
texts = {m.get("text") for m in mb}
raise SystemExit(0 if ids_b == ids_c and texts >= {"keep-1", "keep-2"} else 1)
PY
  then
    matched=1
    break
  fi
  sleep 0.1
done
[[ "$matched" -eq 1 ]] || fail "C tail != new keeper; b=$(curl -sS "http://${listen_b}/tail") c=$(curl -sS "http://${listen_c}/tail")"

for p in "$pid_b" "$pid_c"; do kill "$p" 2>/dev/null || true; done
for p in "$pid_b" "$pid_c"; do wait "$p" 2>/dev/null || true; done
pid_b=""; pid_c=""

echo "keeper_leave_test OK"
