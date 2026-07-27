#!/usr/bin/env bash
# P083: Android APK/AAB → dist/
# Usage: ./scripts/build_android_apk.sh
# Optional: DIST=/tmp/out ./scripts/build_android_apk.sh
# Requires: Flutter, Android SDK (ANDROID_HOME or brew cask android-commandlinetools), JDK 17–25.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_android_apk FAIL: $*" >&2
  exit 1
}

export PATH="/opt/homebrew/share/flutter/bin:${PATH:-}"
command -v flutter >/dev/null 2>&1 || fail "flutter not on PATH"

if [[ -z "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" ]]; then
  if [[ -d /opt/homebrew/share/android-commandlinetools ]]; then
    export ANDROID_HOME=/opt/homebrew/share/android-commandlinetools
    export ANDROID_SDK_ROOT="$ANDROID_HOME"
  fi
fi
[[ -n "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" ]] || fail "ANDROID_HOME not set (install Android SDK)"

if [[ -z "${JAVA_HOME:-}" ]]; then
  for j in openjdk@23 openjdk@21 openjdk@17 openjdk; do
    cand="/opt/homebrew/opt/$j/libexec/openjdk.jdk/Contents/Home"
    if [[ -x "$cand/bin/java" ]]; then
      export JAVA_HOME="$cand"
      break
    fi
  done
fi
[[ -n "${JAVA_HOME:-}" ]] || fail "JAVA_HOME not set (need JDK 17–25)"

OUT="${DIST:-$ROOT/dist}"
mkdir -p "$OUT"

echo "flutter build apk --release"
(
  cd apps/dudka
  flutter pub get >/dev/null
  flutter build apk --release
)

APK_SRC="apps/dudka/build/app/outputs/flutter-apk/app-release.apk"
[[ -f "$APK_SRC" ]] || fail "missing $APK_SRC"
cp "$APK_SRC" "$OUT/dudka-android.apk"

echo "flutter build appbundle --release"
(
  cd apps/dudka
  flutter build appbundle --release
)
AAB_SRC="apps/dudka/build/app/outputs/bundle/release/app-release.aab"
[[ -f "$AAB_SRC" ]] || fail "missing $AAB_SRC"
cp "$AAB_SRC" "$OUT/dudka-android.aab"

cat >"$OUT/BUILD-ANDROID.md" <<'EOF'
# Android build (P083)

```bash
./scripts/build_android_apk.sh
# → dist/dudka-android.apk
# → dist/dudka-android.aab
```

Install on a family phone (sideload):

```bash
adb install -r dist/dudka-android.apk
```

## Engine / sidecar note

macOS/Windows bundle `dudkad` next to the GUI binary. On Android the same subprocess
sidecar needs an NDK-built `libdudkad.so` / extracted binary (not packaged in this MVP APK).
For LAN demo: run `dudkad` on a desktop in the same Wi‑Fi and point the app at it via
`--dart-define=DUDKA_ENGINE=http://<lan-ip>:<port>` (dev) or use a phone that shares
the apartment LAN with a desktop engine until mobile embed lands.
EOF

echo "OK"
echo "  $OUT/dudka-android.apk"
echo "  $OUT/dudka-android.aab"
