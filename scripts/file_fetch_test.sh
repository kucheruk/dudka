#!/usr/bin/env bash
# Task-level contract for P051: chunk download from source → full file on disk.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_fetch_test FAIL: $*" >&2
  exit 1
}

go test ./internal/files/ ./internal/chat/ ./internal/loopback/ -run 'Chunk|FetchFile|FilesFetch' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
bin="$tmpdir/dudkad"
go build -o "$bin" ./cmd/dudkad || fail "go build failed"

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

log_a="$tmpdir/a.log"
log_b="$tmpdir/b.log"
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
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "not ready"

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

# 24-byte payload → multiple 8-byte chunks on the wire when ChunkSize defaults apply;
# engine uses 64KiB default, but still transfers via file_chunk frames (not announce body).
ann="$(python3 - <<'PY'
import base64, hashlib, json
payload = b"p051-chunked-payload-OK!!"
print(json.dumps({
  "name": "payload.bin",
  "mime": "application/octet-stream",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' \
  -d "$ann" >"$tmpdir/ann.json" || fail "announce failed"
file_id="$(python3 - "$tmpdir/ann.json" <<'PY'
import json, sys
d = json.load(open(sys.argv[1]))
m = d["message"]
assert m["type"] == "file_announce"
assert m["file_id"]
# content must not ride on the announce message
assert not m.get("content_b64")
assert not m.get("text")
print(m["file_id"])
PY
)"

# The sender uses the same async API as the GUI. It must receive a named inbox
# file, never the raw extensionless blob keyed only by file_id.
curl -sS --max-time 2 -X POST "http://${listen_a}/files/fetch" \
  -H 'Content-Type: application/json' \
  -d "{\"file_id\":\"${file_id}\",\"wait\":false}" >/dev/null \
  || fail "own async fetch failed to start"
own_done=0
for _ in $(seq 1 40); do
  curl -sS --max-time 1 "http://${listen_a}/files/transfers" >"$tmpdir/own-transfers.json" || true
  if python3 - "$tmpdir/own-transfers.json" "$file_id" <<'PY'
import json, sys
trs = json.load(open(sys.argv[1])).get("transfers") or []
fid = sys.argv[2]
raise SystemExit(0 if any(
    t.get("file_id") == fid and t.get("status") == "done" and t.get("path")
    for t in trs
) else 1)
PY
  then
    own_done=1
    break
  fi
  sleep 0.05
done
[[ "$own_done" -eq 1 ]] || fail "own async fetch did not finish"
python3 - "$tmpdir/own-transfers.json" "$file_id" <<'PY' || fail "own fetch leaked blob path"
import json, pathlib, sys
trs = json.load(open(sys.argv[1])).get("transfers") or []
tr = next(t for t in trs if t.get("file_id") == sys.argv[2])
path = pathlib.Path(tr["path"])
assert path.name == "payload.bin", tr
assert "blobs" not in path.parts, tr
assert path.read_bytes() == b"p051-chunked-payload-OK!!", tr
PY

# Wait until Bob sees announce.
seen=0
for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  if python3 - "$mb" "$file_id" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
fid = sys.argv[2]
for m in data.get("messages") or []:
    if m.get("file_id") == fid and m.get("type") == "file_announce":
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    seen=1
    break
  fi
  sleep 0.05
done
[[ "$seen" -eq 1 ]] || fail "bob missing announce"

curl -sS --max-time 5 -X POST "http://${listen_b}/files/fetch" \
  -H 'Content-Type: application/json' \
  -d "{\"file_id\":\"${file_id}\"}" >"$tmpdir/fetch.json" || fail "fetch failed"
python3 - "$tmpdir/fetch.json" <<'PY' || fail "fetch result bad"
import json, sys, pathlib
d = json.load(open(sys.argv[1]))
path = pathlib.Path(d["path"])
assert path.is_file(), d
assert path.name == "payload.bin", d
data = path.read_bytes()
assert data == b"p051-chunked-payload-OK!!", data
assert d.get("size") == len(data), d
assert d.get("file_id"), d
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_fetch_test OK"
