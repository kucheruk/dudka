#!/usr/bin/env bash
# Task-level contract for P082: Windows runnable artifacts in dist/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_windows_app_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_windows_app.sh ]] || fail "build_windows_app.sh missing"
grep -q 'GOOS=windows' scripts/build_windows_app.sh || fail "must target windows"
grep -q 'build_windows_app' README.md || fail "README must document Windows build"
[[ -d apps/dudka/windows ]] || fail "Flutter windows/ scaffold missing"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
DIST="$tmpdir/dist" ./scripts/build_windows_app.sh || fail "build failed"

exe="$tmpdir/dist/dudkad-windows-amd64.exe"
tui="$tmpdir/dist/dudka-windows-amd64.exe"
[[ -f "$exe" && -s "$exe" ]] || fail "missing $exe"
[[ -f "$tui" && -s "$tui" ]] || fail "missing $tui"
[[ -f "$tmpdir/dist/BUILD-WINDOWS.md" ]] || fail "BUILD-WINDOWS.md missing"

if command -v file >/dev/null 2>&1; then
  file "$exe" | grep -qiE 'PE32|Windows' || fail "engine not a Windows PE: $(file "$exe")"
  file "$tui" | grep -qiE 'PE32|Windows' || fail "TUI not a Windows PE: $(file "$tui")"
fi

echo "build_windows_app_test OK exe=$exe"
