#!/usr/bin/env bash
# P150 contract: Linux has a full Flutter GUI package.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_linux_app_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_linux_app.sh ]] || fail "build script missing"
[[ -d apps/dudka/linux ]] || fail "Flutter Linux scaffold missing"
[[ -f packaging/linux/team.zamoo.dudka.desktop ]] ||
  fail "desktop entry missing"
grep -q 'flutter build linux --release' scripts/build_linux_app.sh ||
  fail "must build Flutter GUI"
grep -q 'Terminal=false' packaging/linux/team.zamoo.dudka.desktop ||
  fail "desktop entry must not open a terminal"
grep -q 'dudka-linux-amd64.deb' README.md ||
  fail "README must document the GUI installer"

if [[ "$(uname -s)" == Linux ]]; then
  tmpdir="$(mktemp -d)"
  trap 'rm -rf "$tmpdir"' EXIT
  DIST="$tmpdir/dist" ./scripts/build_linux_app.sh
  [[ -s "$tmpdir/dist/dudka-linux-amd64.deb" ]] || fail "DEB missing"
  [[ -s "$tmpdir/dist/dudka-linux-amd64.tar.gz" ]] || fail "tar missing"
fi

echo "build_linux_app_test OK"
