#!/usr/bin/env bash
# P090: lab NFR text latency between two dudkad on one host.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() { echo "nfr_latency_test FAIL: $*" >&2; exit 1; }

tmpdir="$(mktemp -d)"
trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$tmpdir"' EXIT

go build -o "$tmpdir/dudkad" ./cmd/dudkad

start_one() {
  local name="$1" dir="$2" listen="$3"
  "$tmpdir/dudkad" -data-dir "$dir" -name "$name" -listen "$listen" \
    -announce-port 0 -session-port 0 -announce-interval 500ms \
    >"$dir/log" 2>&1 &
  local i
  for i in $(seq 1 50); do
    grep -q '^ready ' "$dir/log" 2>/dev/null && return 0
    sleep 0.1
  done
  fail "no ready for $name: $(cat "$dir/log")"
}

mkdir -p "$tmpdir/a" "$tmpdir/b"
start_one "A" "$tmpdir/a" "127.0.0.1:0"
start_one "B" "$tmpdir/b" "127.0.0.1:0"

# Parse listen + session from logs; wire announce via -announce-target is harder post-start.
# Instead: extract peer TCP from session_tcp and dial via POST /scan or seed.
# Simpler path: both use SO_REUSEPORT default — restart with shared announce port.

kill $(jobs -p) 2>/dev/null || true
wait 2>/dev/null || true

ANN=41779
start_shared() {
  local name="$1" dir="$2" listen="$3"
  rm -f "$dir/log"
  "$tmpdir/dudkad" -data-dir "$dir" -name "$name" -listen "$listen" \
    -announce-port "$ANN" -session-port 0 -announce-interval 300ms \
    >"$dir/log" 2>&1 &
  for _ in $(seq 1 50); do
    grep -q '^ready ' "$dir/log" 2>/dev/null && break
    sleep 0.1
  done
  grep -q '^ready ' "$dir/log" || fail "ready $name"
}

mkdir -p "$tmpdir/a" "$tmpdir/b"
start_shared "A" "$tmpdir/a" "127.0.0.1:17901"
start_shared "B" "$tmpdir/b" "127.0.0.1:17902"

# Wait peers visible
for _ in $(seq 1 40); do
  pa=$(curl -sf http://127.0.0.1:17901/peers | python3 -c 'import sys,json; print(len(json.load(sys.stdin).get("peers",[])))')
  pb=$(curl -sf http://127.0.0.1:17902/peers | python3 -c 'import sys,json; print(len(json.load(sys.stdin).get("peers",[])))')
  [[ "$pa" -ge 1 && "$pb" -ge 1 ]] && break
  sleep 0.15
done
[[ "$pa" -ge 1 && "$pb" -ge 1 ]] || fail "peers not visible a=$pa b=$pb"

python3 - <<'PY'
import json, time, urllib.request, statistics
base_a = "http://127.0.0.1:17901"
base_b = "http://127.0.0.1:17902"

def post(url, data):
    req = urllib.request.Request(url, data=json.dumps(data).encode(), headers={"Content-Type":"application/json"}, method="POST")
    with urllib.request.urlopen(req, timeout=2) as r:
        return json.loads(r.read().decode())

def msgs(url):
    with urllib.request.urlopen(url+"/messages", timeout=2) as r:
        return json.loads(r.read().decode()).get("messages", [])

samples = []
for i in range(30):
    token = f"nfr-{i}-{time.time_ns()}"
    t0 = time.perf_counter()
    post(base_a+"/send", {"text": token})
    deadline = t0 + 2.0
    ok = False
    while time.perf_counter() < deadline:
        for m in msgs(base_b):
            if m.get("text") == token:
                samples.append((time.perf_counter() - t0) * 1000)
                ok = True
                break
        if ok:
            break
        time.sleep(0.005)
    if not ok:
        raise SystemExit(f"timeout waiting for {token}")

samples.sort()
def pct(p):
    if not samples: return None
    k = int(round((p/100)*(len(samples)-1)))
    return samples[k]
p50, p95 = pct(50), pct(95)
print(f"nfr_latency_test OK n={len(samples)} p50_ms={p50:.1f} p95_ms={p95:.1f} max_ms={samples[-1]:.1f}")
if p95 > 500:
    raise SystemExit(f"p95 {p95:.1f} > 500 ms (DUD-PRD-120)")
PY
