#!/usr/bin/env bash
# P150: full Linux Flutter GUI → tar.gz + Debian package.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_linux_app FAIL: $*" >&2
  exit 1
}

[[ "$(uname -s)" == Linux ]] || fail "requires Linux"
command -v go >/dev/null 2>&1 || fail "go not on PATH"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v dpkg-deb >/dev/null 2>&1 || fail "dpkg-deb not on PATH"

OUT="${DIST:-$ROOT/dist}"
ARCH="${GOARCH:-amd64}"
VERSION="$(awk '/^version: / { split($2, parts, "+"); print parts[1] }' \
  apps/dudka/pubspec.yaml)"
[[ -n "$VERSION" ]] || fail "version missing in pubspec.yaml"
mkdir -p "$OUT"

(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter build linux --release
)

REL="apps/dudka/build/linux/x64/release/bundle"
[[ -d "$REL" ]] || fail "missing Flutter Linux release bundle"

BUNDLE="$OUT/dudka-linux"
rm -rf "$BUNDLE"
mkdir -p "$BUNDLE"
cp -R "$REL/." "$BUNDLE/"
CGO_ENABLED=0 GOOS=linux GOARCH="$ARCH" go build -trimpath -ldflags='-s -w' \
  -o "$BUNDLE/dudkad" ./cmd/dudkad

TAR="$OUT/dudka-linux-${ARCH}.tar.gz"
rm -f "$TAR"
tar -C "$OUT" -czf "$TAR" dudka-linux

PKG="$(mktemp -d "${TMPDIR:-/tmp}/dudka-deb.XXXXXX")"
trap 'rm -rf "$PKG"' EXIT
mkdir -p \
  "$PKG/DEBIAN" \
  "$PKG/opt/dudka" \
  "$PKG/usr/bin" \
  "$PKG/usr/share/applications" \
  "$PKG/usr/share/icons/hicolor/256x256/apps"
cp -R "$BUNDLE/." "$PKG/opt/dudka/"
ln -s /opt/dudka/dudka "$PKG/usr/bin/dudka"
cp packaging/linux/team.zamoo.dudka.desktop \
  "$PKG/usr/share/applications/team.zamoo.dudka.desktop"
cp apps/dudka/assets/branding/app_icon_master.png \
  "$PKG/usr/share/icons/hicolor/256x256/apps/team.zamoo.dudka.png"
sed \
  -e "s/@VERSION@/$VERSION/" \
  -e "s/@ARCH@/$ARCH/" \
  packaging/linux/control.in >"$PKG/DEBIAN/control"

DEB="$OUT/dudka-linux-${ARCH}.deb"
rm -f "$DEB"
dpkg-deb --build "$PKG" "$DEB" >/dev/null

echo "OK"
echo "  installer: $DEB"
echo "  bundle:    $TAR"
