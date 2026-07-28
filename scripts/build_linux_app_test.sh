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
[[ -f packaging/linux/install.sh.in ]] || fail "curl installer template missing"
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
  [[ -x "$tmpdir/dist/install.sh" ]] || fail "curl installer missing"
  ! grep -q '@VERSION@\\|@DEB_SHA256@' "$tmpdir/dist/install.sh" ||
    fail "curl installer has unresolved placeholders"
  sh -n "$tmpdir/dist/install.sh" || fail "curl installer syntax is invalid"
  DUDKA_INSTALL_DRY_RUN=1 \
    DUDKA_DEB_URL="file://$tmpdir/dist/dudka-linux-amd64.deb" \
    "$tmpdir/dist/install.sh" >/dev/null ||
    fail "curl installer did not verify the built DEB"
  cp "$tmpdir/dist/dudka-linux-amd64.deb" "$tmpdir/dist/corrupt.deb"
  printf 'corrupt' >>"$tmpdir/dist/corrupt.deb"
  if DUDKA_INSTALL_DRY_RUN=1 \
    DUDKA_DEB_URL="file://$tmpdir/dist/corrupt.deb" \
    "$tmpdir/dist/install.sh" >/dev/null 2>&1; then
    fail "curl installer accepted a package with the wrong SHA-256"
  fi
fi

echo "build_linux_app_test OK"
