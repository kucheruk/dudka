#!/usr/bin/env bash
# P060/P061 helper: Flutter macOS skeleton with dudkad (attach or spawn).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"

mode="${1:-attach}" # attach | spawn

go build -o dist/dudkad ./cmd/dudkad
mkdir -p dist

if [[ "$mode" == "spawn" ]]; then
  echo "opening Flutter skeleton; app spawns dist/dudkad"
  cd apps/dudka
  exec flutter run -d macos --dart-define="DUDKAD_BIN=${ROOT}/dist/dudkad"
fi

tmpdir="$(mktemp -d)"
cleanup() {
  [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true
  rm -rf "$tmpdir"
}
trap cleanup EXIT

port="$(python3 - <<'PY'
import socket
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
s.bind(("127.0.0.1", 0))
print(s.getsockname()[1])
s.close()
PY
)"
log="$tmpdir/e.log"
./dist/dudkad -data-dir "$tmpdir/data" -name "ДУДКА" -listen "127.0.0.1:0" \
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
echo "opening Flutter skeleton → GET /me"
cd apps/dudka
exec flutter run -d macos --dart-define="DUDKA_ENGINE=http://${listen}"