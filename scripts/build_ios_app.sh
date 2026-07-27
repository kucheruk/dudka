#!/usr/bin/env bash
# P084: iOS device-arch Flutter build → dist/ (unsigned unless CODESIGN_IDENTITY set).
# Usage: ./scripts/build_ios_app.sh
# Optional: DIST=/tmp/out ./scripts/build_ios_app.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_ios_app FAIL: $*" >&2
  exit 1
}

[[ "$(uname -s)" == Darwin ]] || fail "requires macOS + Xcode"
export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
[[ -d apps/dudka/ios ]] || fail "Flutter ios/ scaffold missing — run flutter create --platforms=ios"

OUT="${DIST:-$ROOT/dist}"
mkdir -p "$OUT"

echo "flutter build ios --release --no-codesign"
(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter build ios --release --no-codesign
)

APP_SRC="apps/dudka/build/ios/iphoneos/Runner.app"
[[ -d "$APP_SRC" ]] || fail "missing $APP_SRC"

APP_DST="$OUT/dudka-ios-Runner.app"
rm -rf "$APP_DST"
cp -R "$APP_SRC" "$APP_DST"

ZIP="$OUT/dudka-ios-unsigned.zip"
rm -f "$ZIP"
(
  cd "$OUT"
  ditto -c -k --sequesterRsrc --keepParent dudka-ios-Runner.app "$(basename "$ZIP")"
)

cp "$ROOT/docs/build-ios.md" "$OUT/BUILD-IOS.md"

if [[ -n "${CODESIGN_IDENTITY:-}" ]]; then
  echo "codesign with $CODESIGN_IDENTITY"
  codesign -f -s "$CODESIGN_IDENTITY" --deep --timestamp=none "$APP_DST" \
    || fail "codesign failed"
fi

echo "OK"
echo "  $APP_DST"
echo "  $ZIP"
echo "  see docs/build-ios.md for ad-hoc / TestFlight"
