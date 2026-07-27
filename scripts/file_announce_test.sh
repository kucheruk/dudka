#!/usr/bin/env bash
# Task-level contract for P050: file-announce in feed without auto-download.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_announce_test FAIL: $*" >&2
  exit 1
}

go test ./internal/chat/ ./internal/loopback/ ./internal/tui/ -run 'FileAnnounce|AnnounceFile' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"

log_a="$tmpdir/a.log"
log_b="$tmpdir/b.log"
"$bin" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$bin" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_b" 2>&1 &
pid_b=$!

listen_a=""
listen_b=""
peer_a=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log_a" 2>/dev/null && grep -q '^ready ' "$log_b" 2>/dev/null; then
    listen_a="$(grep '^listen=' "$log_a" | head -n 1 | sed 's/^listen=//')"
    listen_b="$(grep '^listen=' "$log_b" | head -n 1 | sed 's/^listen=//')"
    peer_a="$(sed -n 's/^peer_id=//p' "$log_a" | head -1)"
    break
  fi
  sleep 0.1
done
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "not ready; a=$(cat "$log_a") b=$(cat "$log_b")"

found=0
for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
try:
    ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
except Exception:
    raise SystemExit(1)
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then
    found=1
    break
  fi
  sleep 0.1
done
[[ "$found" -eq 1 ]] || fail "peers empty"

ann='{"name":"notes.txt","size":11,"mime":"text/plain","hash":"sha256:abc"}'
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' \
  -d "$ann" >"$tmpdir/ann.json" || fail "POST /files/announce failed"
python3 - "$tmpdir/ann.json" <<'PY' || fail "announce response bad"
import json, sys
d = json.load(open(sys.argv[1]))
assert d.get("status") in ("accepted", "queued"), d
m = d.get("message") or {}
assert m.get("type") == "file_announce", m
assert m.get("file_id"), m
assert m.get("name") == "notes.txt", m
assert m.get("size") == 11, m
assert m.get("mime") == "text/plain", m
assert m.get("hash") == "sha256:abc", m
assert not m.get("text"), m
open(sys.argv[1] + ".file_id", "w").write(m["file_id"])
PY
file_id="$(cat "$tmpdir/ann.json.file_id")"

seen=0
for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if python3 - "$mb" "$peer_a" "$file_id" <<'PY'
import json, sys
try:
    data = json.loads(sys.argv[1])
except Exception:
    raise SystemExit(1)
want_peer, want_fid = sys.argv[2], sys.argv[3]
for m in data.get("messages") or []:
    if (m.get("type") == "file_announce"
        and m.get("peer_id") == want_peer
        and m.get("file_id") == want_fid
        and m.get("name") == "notes.txt"
        and m.get("size") == 11
        and m.get("mime") == "text/plain"
        and m.get("hash") == "sha256:abc"):
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    seen=1
    break
  fi
  sleep 0.05
done
[[ "$seen" -eq 1 ]] || fail "Bob missing file announce; messages=$(curl -sS "http://${listen_b}/messages")"

# No auto-download of full file in P050.
code="$(curl -sS -o /dev/null -w '%{http_code}' --max-time 1 "http://${listen_b}/files/${file_id}" || true)"
[[ "$code" != "200" ]] || fail "must not serve file bytes yet; got HTTP $code"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_announce_test OK"
