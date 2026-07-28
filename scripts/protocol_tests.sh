#!/usr/bin/env bash
# Multi-peer protocol suite for the local/CI gate (P045).
# Invoked by ./scripts/check.sh after go test ./...
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "protocol_tests: start"

# Discovery + chat + WAN + TUI exchange — each script spins 2+ peers (or WAN guard).
tests=(
  announce_test.sh
  peers_test.sh
  instance_test.sh
  proto_test.sh
  scan_test.sh
  wan_test.sh
  send_test.sh
  reinstall_persistence_test.sh
  tail_test.sh
  keeper_leave_test.sh
  besteffort_test.sh
  tui_send_test.sh
)

for t in "${tests[@]}"; do
  echo "protocol_tests: $t"
  "./scripts/$t"
done

echo "protocol_tests: OK"
