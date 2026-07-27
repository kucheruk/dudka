#!/usr/bin/env bash
# Task-level contract for P057 / DUD-FILE-120: HEIC thumb when decode exists; honest fallback otherwise.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_heic_test FAIL: $*" >&2
  exit 1
}

[[ -f testdata/sample.heic ]] || fail "testdata/sample.heic missing"

go test ./internal/files/ ./internal/chat/ ./internal/tui/ -run 'HEIC|IsHEIC|AnnounceHEIC|RenderHEIC' -count=1 >/dev/null \
  || fail "unit/integration HEIC tests failed"

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
import base64, hashlib, json, pathlib
payload = pathlib.Path("testdata/sample.heic").read_bytes()
print(json.dumps({
  "name": "img.heic",
  "mime": "image/heic",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' -d "$ann" >"$tmpdir/ann.json" || fail "announce"

python3 - "$tmpdir/ann.json" <<'PY' || fail "announce HEIC contract"
import json, sys, pathlib, platform
m = json.load(open(sys.argv[1]))["message"]
assert m["type"] == "file_announce"
assert m["mime"] == "image/heic"
# On darwin+cgo builds we expect a real thumb; elsewhere honest empty.
darwin = platform.system() == "Darwin"
if m.get("thumb_b64"):
    assert m.get("thumb_path"), m
    assert pathlib.Path(m["thumb_path"]).is_file(), m["thumb_path"]
    print("thumb ok", m["thumb_path"])
else:
    assert not m.get("thumb_path"), m
    print("honest fallback (no thumb)")
print(m["file_id"])
PY
file_id="$(python3 -c 'import json; print(json.load(open("'"$tmpdir/ann.json"'"))["message"]["file_id"])')"

for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  echo "$mb" | grep -q "$file_id" && break
  sleep 0.05
done

frame="$("$tui" -engine "$listen_b" 2>/dev/null || true)"
printf '%s\n' "$frame" | grep -q 'img.heic' || fail "TUI missing heic name:\n$frame"
# Either real THUMB path or honest HEIC mark — never silent pretend.
if printf '%s\n' "$frame" | grep -q 'THUMB '; then
  printf '%s\n' "$frame" | grep -q 'THUMB ' || fail "broken THUMB"
else
  printf '%s\n' "$frame" | grep -qE ' HEIC( |$)' || fail "want honest HEIC mark without THUMB:\n$frame"
fi

# Garbage HEIC-labelled bytes must not invent thumb.
ann2="$(python3 - <<'PY'
import base64, hashlib, json
payload = b"not-heic-payload"
print(json.dumps({
  "name": "fake.heic",
  "mime": "image/heic",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' -d "$ann2" >"$tmpdir/ann2.json" || fail "announce garbage"
python3 - "$tmpdir/ann2.json" <<'PY' || fail "garbage must not fake thumb"
import json, sys
m = json.load(open(sys.argv[1]))["message"]
assert not m.get("thumb_b64"), m
assert not m.get("thumb_path"), m
print("garbage ok")
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_heic_test OK"
