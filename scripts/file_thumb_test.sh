#!/usr/bin/env bash
# Task-level contract for P056 / DUD-FILE-120: jpeg/png/webp thumb in announce + TUI mark; no fake thumb for non-image.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_thumb_test FAIL: $*" >&2
  exit 1
}

go test ./internal/files/ ./internal/chat/ ./internal/tui/ -run 'Thumb|IsThumb|AnnounceImage|PeerReceives|AnnounceNonImage|RenderShows|RenderNoThumb' -count=1 >/dev/null \
  || fail "unit/integration thumb tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
bin="$tmpdir/dudkad"
tui="$tmpdir/dudka"
go build -o "$bin" ./cmd/dudkad || fail "go build dudkad failed"
go build -o "$tui" ./cmd/dudka || fail "go build dudka failed"

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

for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  echo "$pa" | grep -q peer && echo "$pb" | grep -q peer && break
  sleep 0.1
done

ann="$(python3 - <<'PY'
import base64, hashlib, json
# 48×36 JPEG fixture (no Pillow required).
payload = base64.b64decode(
  "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAUDBAQEAwUEBAQFBQUGBwwIBwcHBw8LCwkMEQ8SEhEPERETFhwXExQaFRERGCEYGh0dHx8fExciJCIeJBweHx7/2wBDAQUFBQcGBw4ICA4eFBEUHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh7/wAARCAAkADADASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwDMooor9ePxwKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigD/9k="
)
print(json.dumps({
  "name": "sky.jpg",
  "mime": "image/jpeg",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"

curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' -d "$ann" >"$tmpdir/ann.json" || fail "announce"
python3 - "$tmpdir/ann.json" <<'PY' || fail "announce missing thumb"
import json, sys, base64, pathlib
d = json.load(open(sys.argv[1]))
m = d["message"]
assert m["type"] == "file_announce", m
assert m.get("thumb_b64"), m
assert m.get("thumb_path"), m
assert pathlib.Path(m["thumb_path"]).is_file(), m["thumb_path"]
raw = base64.b64decode(m["thumb_b64"])
assert raw[:2] == b"\xff\xd8", raw[:8]
print(m["file_id"])
PY
file_id="$(python3 -c 'import json; print(json.load(open("'"$tmpdir/ann.json"'"))["message"]["file_id"])')"

for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  echo "$mb" | grep -q "$file_id" && break
  sleep 0.05
done

python3 - "$listen_b" "$file_id" <<'PY' || fail "bob missing thumb"
import json, pathlib, sys, urllib.request
listen, fid = sys.argv[1], sys.argv[2]
raw = urllib.request.urlopen(f"http://{listen}/messages", timeout=2).read()
msgs = json.loads(raw).get("messages") or []
hit = [m for m in msgs if m.get("file_id") == fid]
assert hit, msgs
m = hit[0]
assert m.get("thumb_b64"), m
assert m.get("thumb_path"), m
assert pathlib.Path(m["thumb_path"]).is_file(), m["thumb_path"]
print("bob ok", m["thumb_path"])
PY

frame="$("$tui" -engine "$listen_b" 2>/dev/null || true)"
printf '%s\n' "$frame" | grep -q 'THUMB' || fail "TUI missing THUMB mark:\n$frame"
printf '%s\n' "$frame" | grep -q 'sky.jpg' || fail "TUI missing name:\n$frame"

# Non-image must not invent THUMB.
ann2="$(python3 - <<'PY'
import base64, hashlib, json
payload = b"p056-not-image"
print(json.dumps({
  "name": "notes.txt",
  "mime": "text/plain",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' -d "$ann2" >"$tmpdir/ann2.json" || fail "announce text"
file_id2="$(python3 -c 'import json; print(json.load(open("'"$tmpdir/ann2.json"'"))["message"]["file_id"])')"
python3 - "$tmpdir/ann2.json" <<'PY' || fail "text announce must not have thumb"
import json, sys
m = json.load(open(sys.argv[1]))["message"]
assert not m.get("thumb_b64"), m
assert not m.get("thumb_path"), m
PY

for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  echo "$mb" | grep -q "$file_id2" && break
  sleep 0.05
done
frame2="$("$tui" -engine "$listen_b" 2>/dev/null || true)"
# FEED may still show THUMB for the jpeg row — ensure notes.txt line has no THUMB after the name.
python3 - "$frame2" <<'PY' || fail "notes.txt must not show THUMB"
import sys
frame = sys.argv[1]
for line in frame.splitlines():
    if "notes.txt" in line and "THUMB" in line:
        raise SystemExit(f"false thumb on text: {line}")
print("ok")
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_thumb_test OK"
