#!/usr/bin/env bash
# Task-level contract for P055 / DUD-FILE-130: hash mismatch after download → corrupt error, not success.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "file_hash_test FAIL: $*" >&2
  exit 1
}

go test ./internal/files/ ./internal/chat/ -run 'Hash|Corrupt|Verify|FetchHash' -count=1 >/dev/null \
  || fail "unit/integration hash tests failed"

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

# Source stores real bytes but announce lies about hash → Bob must reject after download.
ann="$(python3 - <<'PY'
import base64, hashlib, json
payload = b"p055-real-bytes-on-disk"
wrong = "sha256:" + hashlib.sha256(b"p055-tampered-expected").hexdigest()
print(json.dumps({
  "name": "corrupt.bin",
  "mime": "application/octet-stream",
  "hash": wrong,
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
echo "$(curl -sS --max-time 1 "http://${listen_b}/messages")" | grep -q "$file_id" \
  || fail "bob missing announce"

set +e
http_code="$(curl -sS --max-time 5 -o "$tmpdir/fetch.body" -w '%{http_code}' \
  -X POST "http://${listen_b}/files/fetch" \
  -H 'Content-Type: application/json' \
  -d "{\"file_id\":\"${file_id}\"}")"
set -e
[[ "$http_code" != "200" ]] || fail "fetch HTTP must fail, got 200: $(cat "$tmpdir/fetch.body")"
body="$(cat "$tmpdir/fetch.body")"
echo "$body" | grep -qE 'повреждён|поврежден' || fail "fetch body missing corrupt text: $body"

python3 - "$listen_b" "$file_id" "$tmpdir/b" <<'PY' || fail "transfer status / inbox check"
import json, pathlib, sys, urllib.request
listen, fid, data = sys.argv[1], sys.argv[2], sys.argv[3]
raw = urllib.request.urlopen(f"http://{listen}/files/transfers", timeout=2).read()
trs = json.loads(raw).get("transfers") or []
hit = [t for t in trs if t.get("file_id") == fid]
assert hit, trs
tr = hit[0]
assert tr.get("status") == "error", tr
assert not tr.get("path"), tr
err = (tr.get("error") or "")
assert ("повреждён" in err) or ("поврежден" in err), err
inbox = pathlib.Path(data) / "inbox"
if inbox.is_dir():
    for p in inbox.rglob("*"):
        if p.is_file() and fid in p.name:
            raise SystemExit(f"corrupt success file remains: {p}")
print("ok")
PY

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "file_hash_test OK"
