#!/usr/bin/env bash
# P081: build macOS Flutter desktop → dist/dudka.app + zip archive.
# Usage: ./scripts/build_macos_app.sh
# Optional: DIST=/tmp/out ./scripts/build_macos_app.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_macos_app FAIL: $*" >&2
  exit 1
}

[[ "$(uname -s)" == Darwin ]] || fail "requires macOS"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v go >/dev/null 2>&1 || fail "go not on PATH"

OUT="${DIST:-$ROOT/dist}"
mkdir -p "$OUT"

ARCH="$(uname -m)"
case "$ARCH" in
  arm64|aarch64) GOARCH=arm64 ;;
  x86_64|amd64) GOARCH=amd64 ;;
  *) fail "unsupported arch $ARCH" ;;
esac

echo "building darwin dudkad ($GOARCH)"
CGO_ENABLED=0 GOOS=darwin GOARCH="$GOARCH" go build -trimpath -ldflags='-s -w' \
  -o "$OUT/dudkad-darwin-${GOARCH}" ./cmd/dudkad

echo "flutter build macos --release"
(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter build macos --release
)

APP_SRC="apps/dudka/build/macos/Build/Products/Release/dudka.app"
[[ -d "$APP_SRC" ]] || fail "flutter did not produce $APP_SRC"

APP_DST="$OUT/dudka.app"
rm -rf "$APP_DST"
cp -R "$APP_SRC" "$APP_DST"
cp "$OUT/dudkad-darwin-${GOARCH}" "$APP_DST/Contents/MacOS/dudkad"
chmod +x "$APP_DST/Contents/MacOS/dudkad"

ZIP="$OUT/dudka-macos.zip"
rm -f "$ZIP"
(
  cd "$OUT"
  ditto -c -k --sequesterRsrc --keepParent dudka.app "$(basename "$ZIP")"
)

echo "OK"
echo "  $APP_DST"
echo "  $ZIP"
echo "  open $APP_DST"
