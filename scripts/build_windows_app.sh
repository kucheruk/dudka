#!/usr/bin/env bash
# P156: full Windows Flutter GUI → one portable ZIP.
# Run on Windows (locally or in GitHub Actions).
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_windows_app FAIL: $*" >&2
  exit 1
}

HOST="$(uname -s)"
if [[ "$HOST" != MINGW* && "$HOST" != MSYS* && "$HOST" != CYGWIN* &&
  "$HOST" != Windows_NT ]]; then
  fail "полный Windows GUI собирается на Windows; запустите desktop-build workflow"
fi

command -v go >/dev/null 2>&1 || fail "go not on PATH"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"
command -v powershell.exe >/dev/null 2>&1 || fail "powershell.exe not on PATH"

OUT="${DIST:-$ROOT/dist}"
ARCH="${GOARCH:-amd64}"
mkdir -p "$OUT"

echo "building hidden Windows engine"
ENGINE="$OUT/dudkad.exe"
CGO_ENABLED=0 GOOS=windows GOARCH="$ARCH" go build -trimpath \
  -ldflags='-s -w -H=windowsgui' -o "$ENGINE" ./cmd/dudkad

echo "building Flutter Windows GUI"
(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter build windows --release
)

REL="apps/dudka/build/windows/x64/runner/Release"
[[ -d "$REL" ]] || REL="apps/dudka/build/windows/runner/Release"
[[ -d "$REL" ]] || fail "missing Flutter Release directory"

BUNDLE="$OUT/dudka-windows"
rm -rf "$BUNDLE"
mkdir -p "$BUNDLE"
cp -R "$REL/." "$BUNDLE/"
mkdir -p "$BUNDLE/internal"
cp "$ENGINE" "$BUNDLE/internal/dudkad.exe"

ZIP="$OUT/dudka-windows-${ARCH}.zip"
rm -f "$ZIP"
BUNDLE_WIN="$(cygpath -w "$BUNDLE")"
ZIP_WIN="$(cygpath -w "$ZIP")"
powershell.exe -NoProfile -Command \
  "Compress-Archive -LiteralPath '$BUNDLE_WIN' -DestinationPath '$ZIP_WIN' -Force"

[[ -s "$ZIP" ]] || fail "portable ZIP missing: $ZIP"

rm -f "$ENGINE"
echo "OK"
echo "  portable: $ZIP"
