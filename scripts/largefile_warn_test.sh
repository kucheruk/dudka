#!/usr/bin/env bash
# Task-level contract for P054: TUI warns when size > 100 MiB; transfer still possible.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "largefile_warn_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -run 'LargeFile|BeginFetch|ParseFetch|RenderLarge' -count=1 >/dev/null \
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

# Metadata-only announce with size > 100 MiB (no blob bytes — warning is pre-start UX).
size=$((100 * 1024 * 1024 + 1))
ann="$(python3 - "$size" <<'PY'
import json, sys
print(json.dumps({
  "name": "huge.bin",
  "size": int(sys.argv[1]),
  "mime": "application/octet-stream",
  "hash": "sha256:p054",
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

frame="$("$tmpdir/dudka" -engine "$listen_b" 2>/dev/null || true)"
printf '%s\n' "$frame" | grep -q 'ВНИМАНИЕ>100МиБ' || fail "TUI missing large-file ВНИМАНИЕ:\n$frame"
printf '%s\n' "$frame" | grep -q 'huge.bin' || fail "missing file name:\n$frame"

# -fetch prints warning to stderr then still attempts start (not a hard block).
set +e
out="$("$tmpdir/dudka" -engine "$listen_b" -fetch "$file_id" 2>"$tmpdir/err.txt")"
rc=$?
set -e
grep -q '100' "$tmpdir/err.txt" || fail "stderr missing 100 MiB warning: $(cat "$tmpdir/err.txt")"
grep -qiE 'ВНИМАНИЕ|WARNING|100 МиБ|100 MiB' "$tmpdir/err.txt" \
  || grep -q 'ВНИМАНИЕ' "$tmpdir/err.txt" \
  || fail "stderr missing warning copy: $(cat "$tmpdir/err.txt")"

# Fetch may fail later (no blob at source) — that is OK for P054; warning must have appeared.
# Ensure we did not hard-refuse solely because of size.
if grep -qiE 'too large|hard.?block|refused by size' "$tmpdir/err.txt"; then
  fail "must not hard-block by size: $(cat "$tmpdir/err.txt")"
fi
# Soft warning must explicitly say transfer is still allowed.
grep -q 'не запрещена' "$tmpdir/err.txt" || fail "warning must say transfer is allowed: $(cat "$tmpdir/err.txt")"
_="$out"; _="$rc"

kill "$pid_a" "$pid_b" 2>/dev/null || true
wait "$pid_a" 2>/dev/null || true
wait "$pid_b" 2>/dev/null || true
pid_a=""; pid_b=""

echo "largefile_warn_test OK"
