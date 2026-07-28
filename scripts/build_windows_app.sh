#!/usr/bin/env bash
# P082: Windows desktop artifacts → dist/
# - Always: cross-compile dudkad.exe (and dudka.exe TUI) for windows/amd64
# - On Windows host: also flutter build windows --release and bundle dudkad.exe
# Usage: ./scripts/build_windows_app.sh
# Optional: DIST=/tmp/out GOARCH=arm64 ./scripts/build_windows_app.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_windows_app FAIL: $*" >&2
  exit 1
}

command -v go >/dev/null 2>&1 || fail "go not on PATH"
OUT="${DIST:-$ROOT/dist}"
ARCH="${GOARCH:-amd64}"
mkdir -p "$OUT"

echo "building Windows engine → $OUT/dudkad-windows-${ARCH}.exe"
CGO_ENABLED=0 GOOS=windows GOARCH="$ARCH" go build -trimpath -ldflags='-s -w' \
  -o "$OUT/dudkad-windows-${ARCH}.exe" ./cmd/dudkad

echo "building Windows TUI → $OUT/dudka-windows-${ARCH}.exe"
CGO_ENABLED=0 GOOS=windows GOARCH="$ARCH" go build -trimpath -ldflags='-s -w' \
  -o "$OUT/dudka-windows-${ARCH}.exe" ./cmd/dudka

cat >"$OUT/BUILD-WINDOWS.md" <<EOF
# Windows build (P082)

## Engine / TUI (from any host with Go)

\`\`\`bash
./scripts/build_windows_app.sh
# → dist/dudkad-windows-amd64.exe
# → dist/dudka-windows-amd64.exe
\`\`\`

## Flutter GUI (requires a Windows machine)

Flutter cannot cross-compile \`windows\` from macOS/Linux.

\`\`\`bat
cd apps\\dudka
flutter pub get
flutter build windows --release
copy ..\\..\\dist\\dudkad-windows-amd64.exe build\\windows\\x64\\runner\\Release\\dudkad.exe
\`\`\`

The release folder is the runnable artifact (dudka.exe + dudkad.exe beside it).
Zip that folder as \`dudka-windows-amd64.zip\` for family install and
auto-update. The update manifest must never point to the standalone TUI exe.

Bundled binary resolution matches macOS: \`resolveBundledDudkadBin\` looks next to the app executable.
EOF

HOST="$(uname -s)"
if [[ "$HOST" == MINGW* || "$HOST" == MSYS* || "$HOST" == CYGWIN* || "$HOST" == Windows_NT ]]; then
  export PATH="/c/flutter/bin:${PATH:-}"
  command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH (Windows GUI build)"
  (
    cd apps/dudka
    flutter pub get >/dev/null
    flutter build windows --release
  ) || fail "flutter build windows failed"
  REL="apps/dudka/build/windows/x64/runner/Release"
  [[ -d "$REL" ]] || REL="apps/dudka/build/windows/runner/Release"
  [[ -d "$REL" ]] || fail "missing Flutter Release dir"
  cp "$OUT/dudkad-windows-${ARCH}.exe" "$REL/dudkad.exe"
  rm -rf "$OUT/dudka-windows"
  mkdir -p "$OUT/dudka-windows"
  cp -R "$REL/." "$OUT/dudka-windows/"
  (
    cd "$OUT"
    if command -v zip >/dev/null 2>&1; then
      rm -f "dudka-windows-${ARCH}.zip"
      zip -qr "dudka-windows-${ARCH}.zip" dudka-windows
    fi
  )
  echo "OK Flutter bundle → $OUT/dudka-windows/"
else
  echo "OK (engine/TUI only on $HOST; Flutter GUI needs Windows — see $OUT/BUILD-WINDOWS.md)"
fi

echo "  $OUT/dudkad-windows-${ARCH}.exe"
echo "  $OUT/dudka-windows-${ARCH}.exe"
