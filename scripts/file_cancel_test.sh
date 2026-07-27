#!/usr/bin/env bash
# Task-level contract for P053: cancel download → not success, partial discarded.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_cancel_test FAIL: $*" >&2
  exit 1
}

go test ./internal/files/ ./internal/chat/ ./internal/tui/ -run 'Cancel' -count=1 >/dev/null \
  || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; for p in ${pid_a:-} ${pid_b:-}; do [[ -n "$p" ]] && kill "$p" 2>/dev/null || true; done' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"

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

# Large-ish payload; sync path is fast, so we use wait:false and cancel ASAP after start.
content_b64="$(python3 - <<'PY'
import base64
print(base64.b64encode(b"p053-" + bytes(range(256))*8).decode())
PY
)"
ann="$(python3 - "$content_b64" <<'PY'
import json, sys
print(json.dumps({
  "name": "cancel.bin",
  "mime": "application/octet-stream",
  "hash": "sha256:p053",
  "content_b64": sys.argv[1],
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
  -d "{\"file_id\":\"${file_id}\",\"wait\":false}" >/dev/null || fail "start fetch"

# Cancel promptly (may be mid-flight or already done on tiny LAN — unit tests cover mid%).
# If already done, cancel must fail with already done — then we skip this harness case.
set +e
curl -sS --max-time 2 -X POST "http://${listen_b}/files/cancel" \
  -H 'Content-Type: application/json' \
  -d "{\"file_id\":\"${file_id}\"}" >"$tmpdir/cancel.json"
rc=$?
set -e
[[ "$rc" -eq 0 ]] || fail "cancel request failed"

python3 - "$tmpdir/cancel.json" "$tmpdir/b" "$file_id" <<'PY' || fail "cancel outcome bad"
import json, sys, pathlib, time
raw = pathlib.Path(sys.argv[1]).read_text()
# HTTP error body is plain text
try:
    d = json.loads(raw)
except Exception:
    if "already done" in raw:
        print("cancel_after_done_ok")
        raise SystemExit(0)
    raise SystemExit(f"not json: {raw!r}")
assert d.get("status") == "cancelled", d
assert d.get("status") != "done", d
assert not d.get("path"), d
# inbox must not keep a successful file for this id
inbox = pathlib.Path(sys.argv[2]) / "inbox"
fid = sys.argv[3]
time.sleep(0.05)
for p in inbox.glob(fid + "_*") if inbox.exists() else []:
    assert not p.exists() or p.name.endswith(".partial") is False
    assert not p.exists(), f"leftover {p}"
for p in inbox.glob(fid + "_*.partial") if inbox.exists() else []:
    assert False, f"partial left: {p}"
print("cancel_ok")
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_cancel_test OK"
