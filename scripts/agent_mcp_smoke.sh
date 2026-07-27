#!/usr/bin/env bash
# P115: human + agent stub exchange text; triple-prefix visible in feed.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
tmpdir="$(mktemp -d)"
trap 'kill $(jobs -p) 2>/dev/null || true; rm -rf "$tmpdir"' EXIT
go build -o "$tmpdir/dudkad" ./cmd/dudkad
go build -o "$tmpdir/dudka-mcp" ./cmd/dudka-mcp
ANN=41782
"$tmpdir/dudkad" -data-dir "$tmpdir/human" -name "Маша" -listen 127.0.0.1:17941 \
  -announce-port "$ANN" -session-port 0 -announce-interval 200ms >"$tmpdir/h.log" 2>&1 &
"$tmpdir/dudkad" -data-dir "$tmpdir/agent" -name "Codex·gpt-5·mac-mini" -agent \
  -listen 127.0.0.1:17942 -announce-port "$ANN" -session-port 0 -announce-interval 200ms >"$tmpdir/a.log" 2>&1 &
for _ in $(seq 1 50); do
  pa=$(curl -sf http://127.0.0.1:17941/peers | python3 -c 'import sys,json; print(len(json.load(sys.stdin).get("peers",[])))' 2>/dev/null || echo 0)
  [[ "$pa" -ge 1 ]] && break
  sleep 0.15
done
[[ "$pa" -ge 1 ]] || { echo "agent_mcp_smoke FAIL peers"; cat "$tmpdir/h.log" "$tmpdir/a.log"; exit 1; }
# agent → chat via MCP tool
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dudka_send","arguments":{"text":"привет от агента"}}}' \
  | "$tmpdir/dudka-mcp" -engine http://127.0.0.1:17942 >/tmp/mcp-out.txt
grep -q 'accepted\|queued' /tmp/mcp-out.txt || { echo "mcp send fail"; cat /tmp/mcp-out.txt; exit 1; }
# human → chat
curl -sf -X POST http://127.0.0.1:17941/send -d '{"text":"привет от Маши"}' >/dev/null
# agent inbox via MCP
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' \
  '{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"dudka_inbox","arguments":{}}}' \
  | "$tmpdir/dudka-mcp" -engine http://127.0.0.1:17942 | tee /tmp/mcp-inbox.txt | grep -q 'привет от Маши'
# feed shows agent triple prefix
curl -sf http://127.0.0.1:17941/messages | grep -q 'Codex·gpt-5·mac-mini'
# peer marked agent
curl -sf http://127.0.0.1:17941/peers | python3 -c 'import sys,json; ps=json.load(sys.stdin)["peers"];
assert any(p.get("is_agent") for p in ps), ps'
echo "agent_mcp_smoke OK"
