#!/usr/bin/env bash
# Task-level contract for P084: iOS build script + docs.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_ios_app_test FAIL: $*" >&2
  exit 1
}

[[ -x scripts/build_ios_app.sh ]] || fail "build_ios_app.sh missing"
grep -q 'flutter build ios' scripts/build_ios_app.sh || fail "must call flutter build ios"
grep -q 'build_ios_app' README.md || fail "README must document iOS build"
[[ -f docs/build-ios.md ]] || fail "docs/build-ios.md missing"
grep -qi 'TestFlight' docs/build-ios.md || fail "TestFlight path missing"
grep -qi 'Ad-hoc\|ad-hoc\|Ad hoc' docs/build-ios.md || fail "ad-hoc path missing"
[[ -d apps/dudka/ios ]] || fail "ios/ scaffold missing"

if [[ "$(uname -s)" != Darwin ]]; then
  echo "build_ios_app_test SKIP (not Darwin) — docs+script OK"
  exit 0
fi

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
DIST="$tmpdir/dist" ./scripts/build_ios_app.sh || fail "build failed"

app="$tmpdir/dist/dudka-ios-Runner.app"
zip="$tmpdir/dist/dudka-ios-unsigned.zip"
[[ -d "$app" ]] || fail "missing $app"
[[ -f "$app/Info.plist" ]] || fail "not an app bundle"
[[ -f "$zip" ]] || fail "missing $zip"
[[ -f "$tmpdir/dist/BUILD-IOS.md" ]] || fail "BUILD-IOS.md missing"

echo "build_ios_app_test OK app=$app zip=$zip"
