#!/usr/bin/env bash
# Task-level contract for P052: download progress 0–100% in API (and TUI render).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_progress_test FAIL: $*" >&2
  exit 1
}

go test ./internal/files/ ./internal/chat/ ./internal/loopback/ ./internal/tui/ \
  -run 'Percent|Progress|Transfer|RenderFileAnnounceShows' -count=1 >/dev/null \
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
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Alice" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 150ms >"$log_a" 2>&1 &
pid_a=$!
"$tmpdir/dudkad" -data-dir "$tmpdir/b" -name "Bob" -listen "127.0.0.1:0" \
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
  if python3 - "$pa" "$pb" <<'PY'
import json, sys
ja, jb = json.loads(sys.argv[1]), json.loads(sys.argv[2])
raise SystemExit(0 if (ja.get("peers") and jb.get("peers")) else 1)
PY
  then break; fi
  sleep 0.1
done

# ~6KiB payload so async fetch has visible mid percents on default 64KiB chunks? 
# Use content that still works; progress ticks at least 0→100. For mid%, rely on unit tests;
# here assert transfers reach 100% and TUI frame shows percent.
ann="$(python3 - <<'PY'
import base64, hashlib, json
payload = b"p052-" + b"x"*200
print(json.dumps({
  "name": "prog.bin",
  "mime": "application/octet-stream",
  "hash": "sha256:" + hashlib.sha256(payload).hexdigest(),
  "content_b64": base64.b64encode(payload).decode(),
}))
PY
)"
curl -sS --max-time 2 -X POST "http://${listen_a}/files/announce" \
  -H 'Content-Type: application/json' -d "$ann" >"$tmpdir/ann.json" || fail "announce"
file_id="$(python3 -c 'import json; print(json.load(open("'"$tmpdir/ann.json"'"))["message"]["file_id"])')"

for _ in $(seq 1 40); do
  mb="$(curl -sS --max-time 1 "http://${listen_b}/messages" || true)"
  echo "$mb" | grep -q "$file_id" && break
  sleep 0.05
done

curl -sS --max-time 2 -X POST "http://${listen_b}/files/fetch" \
  -H 'Content-Type: application/json' \
  -d "{\"file_id\":\"${file_id}\",\"wait\":false}" >"$tmpdir/start.json" || fail "start fetch"
python3 - "$tmpdir/start.json" <<'PY' || fail "start response"
import json, sys
d = json.load(open(sys.argv[1]))
assert d.get("file_id"), d
assert d.get("status") == "downloading" or d.get("percent") == 0 or d.get("status") == "done", d
PY

saw100=0
for _ in $(seq 1 100); do
  tr="$(curl -sS --max-time 1 "http://${listen_b}/files/transfers" || true)"
  if python3 - "$tr" "$file_id" <<'PY'
import json, sys
data = json.loads(sys.argv[1])
fid = sys.argv[2]
for t in data.get("transfers") or []:
    if t.get("file_id") == fid and t.get("percent") == 100 and t.get("status") == "done":
        raise SystemExit(0)
raise SystemExit(1)
PY
  then
    saw100=1
    break
  fi
  sleep 0.05
done
[[ "$saw100" -eq 1 ]] || fail "transfers never reached 100%: $(curl -sS "http://${listen_b}/files/transfers")"

# TUI frame after done should show 100%
frame="$("$tmpdir/dudka" -engine "$listen_b" 2>/dev/null || true)"
printf '%s\n' "$frame" | grep -q '100%' || fail "TUI missing 100%%:\n$frame"
printf '%s\n' "$frame" | grep -q 'prog.bin' || fail "TUI missing file name:\n$frame"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_progress_test OK"
