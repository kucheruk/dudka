#!/usr/bin/env bash
# P060 helper: start dudkad (subprocess) + Flutter spike showing GET /me.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

tmpdir="$(mktemp -d)"
cleanup() {
  [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true
  rm -rf "$tmpdir"
}
trap cleanup EXIT

go build -o "$tmpdir/dudkad" ./cmd/dudkad
port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
log="$tmpdir/e.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/data" -name "Spike" -listen "127.0.0.1:0" \
  -announce-port "$port" -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]]

echo "dudkad listen=$listen (pid=$pid)"
echo "opening Flutter spike → GET /me"
cd apps/dudka
exec flutter run -d macos --dart-define="DUDKA_ENGINE=http://${listen}"
