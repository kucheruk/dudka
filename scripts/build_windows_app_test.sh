#!/usr/bin/env bash
# P149 contract: Windows delivery is a full GUI bundle and one installer.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_windows_app_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_windows_app.sh ]] || fail "build script missing"
[[ -f packaging/windows/dudka.iss ]] || fail "Inno Setup recipe missing"
[[ -f .github/workflows/desktop-build.yml ]] || fail "Windows CI build missing"
grep -q 'flutter build windows --release' scripts/build_windows_app.sh ||
  fail "must build Flutter GUI"
grep -q 'windowsgui' scripts/build_windows_app.sh ||
  fail "bundled engine must not open a console"
grep -q 'PrivilegesRequired=lowest' packaging/windows/dudka.iss ||
  fail "installer must work without administrator rights"
grep -q 'dudka-windows-amd64-setup.exe' README.md ||
  fail "README must point to one installer"
! grep -q 'dudka-windows-amd64.exe + dist/dudkad' README.md ||
  fail "README still advertises raw console executables"

case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN* | Windows_NT)
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  DIST="$tmpdir/dist" ./scripts/build_windows_app.sh
  [[ -s "$tmpdir/dist/dudka-windows-amd64-setup.exe" ]] ||
    fail "installer missing"
  [[ -s "$tmpdir/dist/dudka-windows-amd64.zip" ]] ||
    fail "update ZIP missing"
  ;;
esac

echo "build_windows_app_test OK"
