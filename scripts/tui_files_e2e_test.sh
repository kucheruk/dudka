#!/usr/bin/env bash
# Task-level contract for P058: TUI↔TUI — image with thumb + arbitrary binary arrive end-to-end.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tui_files_e2e_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -run 'DetectMIME|AnnouncePath|ParseAnnounce' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

# Fixtures: JPEG image + arbitrary binary.
python3 - <<'PY' >"$tmpdir/pic.jpg"
import base64, sys
sys.stdout.buffer.write(base64.b64decode(
  "/9j/4AAQSkZJRgABAQAAAQABAAD/2wBDAAUDBAQEAwUEBAQFBQUGBwwIBwcHBw8LCwkMEQ8SEhEPERETFhwXExQaFRERGCEYGh0dHx8fExciJCIeJBweHx7/2wBDAQUFBQcGBw4ICA4eFBEUHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh4eHh7/wAARCAAkADADASIAAhEBAxEB/8QAHwAAAQUBAQEBAQEAAAAAAAAAAAECAwQFBgcICQoL/8QAtRAAAgEDAwIEAwUFBAQAAAF9AQIDAAQRBRIhMUEGE1FhByJxFDKBkaEII0KxwRVS0fAkM2JyggkKFhcYGRolJicoKSo0NTY3ODk6Q0RFRkdISUpTVFVWV1hZWmNkZWZnaGlqc3R1dnd4eXqDhIWGh4iJipKTlJWWl5iZmqKjpKWmp6ipqrKztLW2t7i5usLDxMXGx8jJytLT1NXW19jZ2uHi4+Tl5ufo6erx8vP09fb3+Pn6/8QAHwEAAwEBAQEBAQEBAQAAAAAAAAECAwQFBgcICQoL/8QAtREAAgECBAQDBAcFBAQAAQJ3AAECAxEEBSExBhJBUQdhcRMiMoEIFEKRobHBCSMzUvAVYnLRChYkNOEl8RcYGRomJygpKjU2Nzg5OkNERUZHSElKU1RVVldYWVpjZGVmZ2hpanN0dXZ3eHl6goOEhYaHiImKkpOUlZaXmJmaoqOkpaanqKmqsrO0tba3uLm6wsPExcbHyMnK0tPU1dbX2Nna4uPk5ebn6Onq8vP09fb3+Pn6/9oADAMBAAIRAxEAPwDMooor9ePxwKKKKACiiigAooooAKKKKACiiigAooooAKKKKACiiigD/9k="
))
PY
printf 'p058-binary-\x00\x01\x02\xff-payload' >"$tmpdir/payload.bin"

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

log_a="$tmpdir/a.log"; log_b="$tmpdir/b.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Аня" -listen "127.0.0.1:0" \
  -announce-port "$port_a" -announce-target "127.0.0.1:${port_b}" \
  -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "Боря" -listen "127.0.0.1:0" \
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
[[ -n "$listen_a" && -n "$listen_b" ]] || fail "engines not ready"

for _ in $(seq 1 60); do
  pa="$(curl -sS --max-time 1 "http://${listen_a}/peers" || true)"
  pb="$(curl -sS --max-time 1 "http://${listen_b}/peers" || true)"
  echo "$pa" | grep -q peer && echo "$pb" | grep -q peer && break
  sleep 0.1
done

# --- Image with preview: Alice TUI announces → Bob TUI sees ПРЕВЬЮ → Bob TUI fetches ---
ann_out="$("$tmpdir/dudka" -engine "$listen_a" -announce "$tmpdir/pic.jpg" 2>"$tmpdir/ann_img.err")"
file_img="$(grep -E 'announced file_id=' "$tmpdir/ann_img.err" | sed -E 's/.*file_id=([^ ]+).*/\1/')"
[[ -n "$file_img" ]] || fail "missing image file_id: $(cat "$tmpdir/ann_img.err")"
printf '%s\n' "$ann_out" | grep -q 'pic.jpg' || fail "Alice frame missing pic.jpg:\n$ann_out"
printf '%s\n' "$ann_out" | grep -q 'ПРЕВЬЮ' || fail "Alice frame missing ПРЕВЬЮ:\n$ann_out"

seen=0
for _ in $(seq 1 40); do
  frame_b="$("$tmpdir/dudka" -engine "$listen_b")"
  if printf '%s\n' "$frame_b" | grep -q "$file_img" && printf '%s\n' "$frame_b" | grep -q 'ПРЕВЬЮ'; then
    seen=1
    break
  fi
  sleep 0.1
done
[[ "$seen" -eq 1 ]] || fail "Bob TUI missing image ПРЕВЬЮ: $($tmpdir/dudka -engine "$listen_b")"

"$tmpdir/dudka" -engine "$listen_b" -fetch "$file_img" >/dev/null 2>"$tmpdir/fetch_img.err" \
  || fail "Bob image fetch failed: $(cat "$tmpdir/fetch_img.err")"

python3 - "$listen_b" "$file_img" "$tmpdir/pic.jpg" <<'PY' || fail "image bytes/path check"
import hashlib, json, pathlib, sys, urllib.request
listen, fid, src = sys.argv[1], sys.argv[2], sys.argv[3]
raw = urllib.request.urlopen(f"http://{listen}/files/transfers", timeout=2).read()
trs = json.loads(raw).get("transfers") or []
hit = [t for t in trs if t.get("file_id") == fid]
assert hit, trs
tr = hit[0]
assert tr.get("status") == "done", tr
assert tr.get("percent") == 100, tr
path = pathlib.Path(tr["path"])
assert path.is_file(), tr
got = path.read_bytes()
want = pathlib.Path(src).read_bytes()
assert got == want, (len(got), len(want))
assert hashlib.sha256(got).hexdigest() == hashlib.sha256(want).hexdigest()
print("image ok", path)
PY

# --- Arbitrary binary: Alice TUI announces → Bob sees ФАЙЛ without ПРЕВЬЮ → fetch ---
"$tmpdir/dudka" -engine "$listen_a" -announce "$tmpdir/payload.bin" >/dev/null 2>"$tmpdir/ann_bin.err" \
  || fail "binary announce failed: $(cat "$tmpdir/ann_bin.err")"
file_bin="$(grep -E 'announced file_id=' "$tmpdir/ann_bin.err" | sed -E 's/.*file_id=([^ ]+).*/\1/')"
[[ -n "$file_bin" ]] || fail "missing binary file_id"

seen=0
for _ in $(seq 1 40); do
  frame_b="$("$tmpdir/dudka" -engine "$listen_b")"
  if printf '%s\n' "$frame_b" | grep -q 'payload.bin' && printf '%s\n' "$frame_b" | grep -q "$file_bin"; then
    # The ФАЙЛ line for payload.bin must not carry ПРЕВЬЮ.
    if python3 - "$frame_b" "$file_bin" <<'PY'
import sys
frame, fid = sys.argv[1], sys.argv[2]
for line in frame.splitlines():
    if fid in line and "payload.bin" in line:
        if "ПРЕВЬЮ" in line:
            raise SystemExit(1)
        raise SystemExit(0)
raise SystemExit(2)
PY
    then
      seen=1
      break
    fi
  fi
  sleep 0.1
done
[[ "$seen" -eq 1 ]] || fail "Bob TUI binary announce missing or false ПРЕВЬЮ"

# Fetch via compose /fetch (TUI path), not only -fetch flag.
printf '/fetch %s\n' "$file_bin" | "$tmpdir/dudka" -engine "$listen_b" -watch -interval 10s >/dev/null 2>&1 || true
# Poll until done (watch may exit after stdin EOF before transfer finishes).
for _ in $(seq 1 60); do
  st="$(curl -sS --max-time 1 "http://${listen_b}/files/transfers" || true)"
  if python3 - "$st" "$file_bin" <<'PY'
import json, sys
trs = json.loads(sys.argv[1]).get("transfers") or []
fid = sys.argv[2]
for t in trs:
    if t.get("file_id") == fid and t.get("status") == "done" and t.get("path"):
        raise SystemExit(0)
raise SystemExit(1)
PY
  then break; fi
  # re-kick if needed
  curl -sS --max-time 2 -X POST "http://${listen_b}/files/fetch" \
    -H 'Content-Type: application/json' \
    -d "{\"file_id\":\"${file_bin}\",\"wait\":false}" >/dev/null 2>&1 || true
  sleep 0.1
done

python3 - "$listen_b" "$file_bin" "$tmpdir/payload.bin" <<'PY' || fail "binary bytes check"
import json, pathlib, sys, urllib.request
listen, fid, src = sys.argv[1], sys.argv[2], sys.argv[3]
raw = urllib.request.urlopen(f"http://{listen}/files/transfers", timeout=2).read()
trs = json.loads(raw).get("transfers") or []
hit = [t for t in trs if t.get("file_id") == fid]
assert hit and hit[0].get("status") == "done", hit
got = pathlib.Path(hit[0]["path"]).read_bytes()
want = pathlib.Path(src).read_bytes()
assert got == want, (got, want)
print("binary ok", hit[0]["path"])
PY

# Final Bob TUI frame shows both names and image still has ПРЕВЬЮ mark in feed.
frame_final="$("$tmpdir/dudka" -engine "$listen_b")"
printf '%s\n' "$frame_final" | grep -q 'pic.jpg' || fail "final missing pic.jpg"
printf '%s\n' "$frame_final" | grep -q 'payload.bin' || fail "final missing payload.bin"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "tui_files_e2e_test OK"
