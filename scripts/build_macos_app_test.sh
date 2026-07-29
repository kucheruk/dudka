#!/usr/bin/env bash
# Task-level contract for P081: macOS Flutter .app (or archive) in dist/.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "build_macos_app_test FAIL: $*" >&2
  exit 1
}

[[ "$(uname -s)" == Darwin ]] || fail "P081 macOS build requires Darwin host"
[[ -x scripts/build_macos_app.sh ]] || fail "scripts/build_macos_app.sh missing"
grep -q 'flutter build macos' scripts/build_macos_app.sh || fail "script must call flutter build macos"
grep -q 'build_macos_app' README.md || fail "README must document ./scripts/build_macos_app.sh"
grep -q 'resolveBundledDudkadBin' apps/dudka/lib/main.dart apps/dudka/lib/engine/bundle.dart \
  || fail "app must resolve bundled dudkad (P081)"
[[ -f apps/dudka/lib/engine/bundle.dart ]] || fail "bundle.dart missing"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT
DIST="$tmpdir/dist" ./scripts/build_macos_app.sh || fail "build_macos_app.sh failed"

app="$tmpdir/dist/dudka.app"
zip="$tmpdir/dist/dudka-macos-universal.zip"
dmg="$tmpdir/dist/dudka-macos-universal.dmg"
[[ -d "$app" ]] || fail "missing $app"
[[ -f "$app/Contents/Info.plist" ]] || fail "not a macOS app bundle (no Info.plist)"
[[ -x "$app/Contents/MacOS/dudka" ]] || fail "missing app executable Contents/MacOS/dudka"
[[ -x "$app/Contents/MacOS/dudkad" ]] || fail "bundled dudkad missing next to app binary"
[[ -f "$zip" ]] || fail "missing zip archive $zip"
[[ -f "$dmg" ]] || fail "missing disk image $dmg"
if grep -q 'com.apple.security.app-sandbox' apps/dudka/macos/Runner/Release.entitlements; then
  fail "direct-update build cannot use App Sandbox"
fi
codesign --verify --deep --strict "$app" \
  || fail "app bundle signature is invalid after embedding dudkad"
bundle_version="$(/usr/libexec/PlistBuddy -c 'Print :CFBundleShortVersionString' "$app/Contents/Info.plist")"
expected_version="$(sed -n 's/^version: \([0-9][0-9.]*\)+[0-9][0-9]*$/\1/p' apps/dudka/pubspec.yaml)"
[[ "$bundle_version" == "$expected_version" ]] \
  || fail "bundle version is $bundle_version, want $expected_version"
file "$app/Contents/MacOS/dudkad" | grep -q 'universal binary' \
  || fail "bundled dudkad must contain Intel and Apple Silicon slices"

# Smoke: bundled engine prints ready (does not need GUI).
"$app/Contents/MacOS/dudkad" -data-dir "$tmpdir/eng" -name "Pack" -listen "127.0.0.1:0" \
  >"$tmpdir/eng.log" 2>&1 &
pid=$!
ok=0
for _ in $(seq 1 50); do
  if grep -q '^ready ' "$tmpdir/eng.log" 2>/dev/null; then ok=1; break; fi
  sleep 0.1
done
kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true
[[ "$ok" -eq 1 ]] || fail "bundled dudkad did not become ready:\n$(cat "$tmpdir/eng.log")"

echo "build_macos_app_test OK app=$app zip=$zip dmg=$dmg"
