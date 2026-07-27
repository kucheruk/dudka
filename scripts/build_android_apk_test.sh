#!/usr/bin/env bash
# Task-level contract for P083: Android APK/AAB in dist/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_android_apk_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_android_apk.sh ]] || fail "build_android_apk.sh missing"
grep -q 'flutter build apk' scripts/build_android_apk.sh || fail "script must build apk"
grep -q 'build_android_apk' README.md || fail "README must document Android build"
[[ -d apps/dudka/android ]] || fail "Flutter android/ scaffold missing"
[[ -f apps/dudka/android/app/build.gradle.kts || -f apps/dudka/android/app/build.gradle ]] \
  || fail "android app gradle missing"

# Skip full Gradle on hosts without SDK (CI may not have Android).
if [[ -z "${ANDROID_HOME:-}${ANDROID_SDK_ROOT:-}" && ! -d /opt/homebrew/share/android-commandlinetools ]]; then
  echo "build_android_apk_test SKIP (no Android SDK) — scaffold+script OK"
  exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
DIST="$tmpdir/dist" ./scripts/build_android_apk.sh || fail "build failed"

apk="$tmpdir/dist/dudka-android.apk"
aab="$tmpdir/dist/dudka-android.aab"
[[ -f "$apk" && -s "$apk" ]] || fail "missing $apk"
[[ -f "$aab" && -s "$aab" ]] || fail "missing $aab"
# ZIP local file header
python3 - <<PY
import struct,sys
p="$apk"
with open(p,"rb") as f:
  sig=f.read(4)
assert sig==b"PK\x03\x04", f"apk not a zip: {sig!r}"
print("apk zip ok")
PY
[[ -f "$tmpdir/dist/BUILD-ANDROID.md" ]] || fail "BUILD-ANDROID.md missing"

echo "build_android_apk_test OK apk=$apk aab=$aab"
