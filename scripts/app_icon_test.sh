#!/usr/bin/env bash
# DUD-UI-172: selected four-block mark, deterministic platform assets.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "app_icon_test FAIL: $*" >&2
  exit 1
}

SOURCE="apps/dudka/assets/branding/app_icon_source.png"
MASTER="apps/dudka/assets/branding/app_icon_master.png"
SMALL="apps/dudka/macos/Runner/Assets.xcassets/AppIcon.appiconset/app_icon_16.png"
HASHES="apps/dudka/assets/branding/app_icon_assets.sha256"

[[ -f "$SOURCE" ]] || fail "selected source missing"
[[ -f "$MASTER" ]] || fail "PNG preview missing"
[[ -f "$HASHES" ]] || fail "platform asset hashes missing"
[[ -x scripts/generate_app_icons.sh ]] || fail "generator missing"

expected_source_sha="64e92fbc2a69bfc96103678f2d956466d54a2229d376ad90bb0117e70c387835"
actual_source_sha="$(shasum -a 256 "$SOURCE" | awk '{print $1}')"
[[ "$actual_source_sha" == "$expected_source_sha" ]] \
  || fail "selected source differs from the owner-approved icon"
shasum -a 256 --check "$HASHES" >/dev/null \
  || fail "one or more platform icons differ from the approved generated set"

if ! command -v magick >/dev/null 2>&1; then
  echo "app_icon_test OK approved asset hashes (ImageMagick unavailable; regeneration skipped)"
  exit 0
fi

./scripts/generate_app_icons.sh >/dev/null
shasum -a 256 --check "$HASHES" >/dev/null \
  || fail "generator output differs from the approved generated set"

[[ "$(magick identify -format '%wx%h' "$MASTER")" == "1024x1024" ]] \
  || fail "master must be 1024x1024"
[[ "$(magick identify -format '%z' "$MASTER")" == "8" ]] \
  || fail "master must be 8-bit"
[[ "$(magick identify -format '%[opaque]' "$MASTER")" == "True" ]] \
  || fail "master must be opaque"
[[ "$(magick identify -format '%wx%h' "$SMALL")" == "16x16" ]] \
  || fail "missing 16 px proof"

[[ "$(magick "$MASTER" -format '%[fx:p{285,285}.r>.8&&p{285,285}.g<.4&&p{285,285}.b<.3]' info:)" == "1" ]] \
  || fail "top-left red block missing"
[[ "$(magick "$MASTER" -format '%[fx:p{740,285}.r>.8&&p{740,285}.g>.35&&p{740,285}.g<.75&&p{740,285}.b<.2]' info:)" == "1" ]] \
  || fail "top-right orange block missing"
[[ "$(magick "$MASTER" -format '%[fx:p{285,740}.r>.8&&p{285,740}.g>.8&&p{285,740}.b>.8]' info:)" == "1" ]] \
  || fail "bottom-left off-white block missing"
[[ "$(magick "$MASTER" -format '%[fx:p{740,740}.r>.8&&p{740,740}.g>.65&&p{740,740}.b<.2]' info:)" == "1" ]] \
  || fail "bottom-right yellow block missing"
[[ "$(magick "$MASTER" -format '%[fx:p{512,512}.r<.1&&p{512,512}.g<.1&&p{512,512}.b<.1]' info:)" == "1" ]] \
  || fail "black angular centre missing"

[[ "$(magick identify "apps/dudka/windows/runner/resources/app_icon.ico" | wc -l | tr -d ' ')" -eq 6 ]] \
  || fail "Windows ICO must contain six native sizes"

echo "app_icon_test OK master=$MASTER small=$SMALL"
