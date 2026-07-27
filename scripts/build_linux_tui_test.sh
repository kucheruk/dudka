#!/usr/bin/env bash
# Task-level contract for P080: one-command Linux TUI build → dist/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_linux_tui_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_linux_tui.sh ]] || fail "scripts/build_linux_tui.sh missing or not executable"
grep -q 'GOOS=linux' scripts/build_linux_tui.sh || fail "build script must target GOOS=linux"
grep -q 'dist/' scripts/build_linux_tui.sh || fail "build script must write under dist/"
grep -q 'build_linux_tui' README.md || fail "README must document ./scripts/build_linux_tui.sh"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
# Isolate dist so we do not clobber a developer tree mid-flight.
DIST="$tmpdir/dist" ./scripts/build_linux_tui.sh || fail "build_linux_tui.sh failed"

arch="${GOARCH:-amd64}"
bin="$tmpdir/dist/dudka-linux-${arch}"
eng="$tmpdir/dist/dudkad-linux-${arch}"
[[ -f "$bin" ]] || fail "missing TUI artifact $bin"
[[ -f "$eng" ]] || fail "missing engine artifact $eng"
[[ -s "$bin" ]] || fail "empty TUI binary"
[[ -s "$eng" ]] || fail "empty engine binary"

# Cross-built Linux ELF (or native ELF when already on Linux).
if command -v file >/dev/null 2>&1; then
  file "$bin" | grep -qiE 'ELF|executable' || fail "TUI not an executable: $(file "$bin")"
  file "$eng" | grep -qiE 'ELF|executable' || fail "engine not an executable: $(file "$eng")"
fi

# Must not require cgo for the Linux TUI package path.
CGO_ENABLED=0 GOOS=linux GOARCH="$arch" go list -f '{{.ImportPath}}' ./cmd/dudka >/dev/null \
  || fail "cmd/dudka not listable with CGO_ENABLED=0"

echo "build_linux_tui_test OK bin=$bin"
