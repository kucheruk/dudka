#!/usr/bin/env bash
# P156 contract: Windows delivery is one portable GUI archive.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_windows_app_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_windows_app.sh ]] || fail "build script missing"
[[ ! -e packaging/windows/dudka.iss ]] || fail "installer recipe must be removed"
[[ -f .github/workflows/desktop-build.yml ]] || fail "Windows CI build missing"
grep -q 'flutter build windows --release' scripts/build_windows_app.sh ||
  fail "must build Flutter GUI"
grep -q 'windowsgui' scripts/build_windows_app.sh ||
  fail "bundled engine must not open a console"
grep -q 'dudka-windows-amd64.zip' README.md ||
  fail "README must point to the portable ZIP"
! grep -Eq 'setup\.exe|Inno Setup|innosetup' \
  scripts/build_windows_app.sh .github/workflows/desktop-build.yml README.md \
  docs/specs/ui.md docs/specs/product.md docs/family-install.md ||
  fail "Windows installer is still part of the current delivery contract"
! grep -q 'dudka-windows-amd64.exe + dist/dudkad' README.md ||
  fail "README still advertises raw console executables"
grep -q 'internal/dudkad.exe' scripts/build_windows_app.sh ||
  fail "engine must be kept in the internal portable directory"

case "$(uname -s)" in
MINGW* | MSYS* | CYGWIN* | Windows_NT)
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  DIST="$tmpdir/dist" ./scripts/build_windows_app.sh
  [[ -s "$tmpdir/dist/dudka-windows-amd64.zip" ]] ||
    fail "portable ZIP missing"
  ZIP_WIN="$(cygpath -w "$tmpdir/dist/dudka-windows-amd64.zip")"
  powershell.exe -NoProfile -Command \
    "\$entries = [IO.Compression.ZipFile]::OpenRead('$ZIP_WIN').Entries.FullName; \
      if (-not (\$entries -contains 'dudka-windows/dudka.exe')) { exit 1 }; \
      if (-not (\$entries -contains 'dudka-windows/internal/dudkad.exe')) { exit 1 }; \
      if (\$entries -contains 'dudka-windows/dudkad.exe') { exit 1 }" ||
    fail "portable ZIP layout is invalid"
  ;;
esac

echo "build_windows_app_test OK"
