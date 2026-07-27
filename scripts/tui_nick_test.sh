#!/usr/bin/env bash
# Task-level contract for P043: TUI /nick — new nick on subsequent messages.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

fail() {
  echo "tui_nick_test FAIL: $*" >&2
  exit 1
}

go test ./internal/tui/ -run 'Nick|SetNick|MentionsNick' -count=1 >/dev/null || fail "unit tests failed"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"; [[ -n "${pid:-}" ]] && kill "$pid" 2>/dev/null || true' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "build dudkad"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "build dudka"

log="$tmpdir/a.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/a" -name "Старый" -listen "127.0.0.1:0" \
  -announce-port 0 -session-port 0 -announce-interval 1h >"$log" 2>&1 &
pid=$!

listen=""
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$log" 2>/dev/null; then
    listen="$(grep '^listen=' "$log" | head -n 1 | sed 's/^listen=//')"
    break
  fi
  sleep 0.1
done
[[ -n "$listen" ]] || fail "not ready: $(cat "$log")"

"$tmpdir/dudka" -engine "$listen" -send "до смены" >/dev/null || fail "send before nick"
"$tmpdir/dudka" -engine "$listen" -nick "НовыйНик" >/dev/null || fail "nick flag"
# Also exercise /nick command path.
printf '/nick ЕщёНовее\nпосле смены\n' | "$tmpdir/dudka" -engine "$listen" -watch -interval 10s >/dev/null \
  || fail "nick via compose"

frame="$("$tmpdir/dudka" -engine "$listen")"
printf '%s\n' "$frame" | grep -q 'ЕщёНовее' || fail "status/me missing new nick:\n$frame"
printf '%s\n' "$frame" | grep -q 'после смены' || fail "missing new message:\n$frame"

# Old message keeps old display_name_at_send; new uses latest nick.
python3 - "$listen" <<'PY' || fail "name_at_send check failed"
import json, urllib.request, sys
listen = sys.argv[1]
with urllib.request.urlopen(f"http://{listen}/messages", timeout=2) as r:
    data = json.load(r)
msgs = data.get("messages") or []
by_text = {m.get("text"): m.get("display_name_at_send") for m in msgs}
assert by_text.get("до смены") == "Старый", by_text
assert by_text.get("после смены") == "ЕщёНовее", by_text
PY

me="$(curl -sS --max-time 1 "http://${listen}/me")"
python3 - "$me" <<'PY' || fail "GET /me nick"
import json, sys
d = json.loads(sys.argv[1])
assert d.get("name") == "ЕщёНовее", d
PY

kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
pid=""

echo "tui_nick_test OK"
