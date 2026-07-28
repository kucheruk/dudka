#!/usr/bin/env bash
# DUD-UI-172: render the selected icon source into all native application sizes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "generate_app_icons FAIL: $*" >&2
  exit 1
}

command -v magick >/dev/null 2>&1 || fail "ImageMagick magick is required"

SOURCE="apps/dudka/assets/branding/app_icon_source.png"
MASTER="apps/dudka/assets/branding/app_icon_master.png"
[[ -f "$SOURCE" ]] || fail "missing $SOURCE"

render() {
  local size="$1"
  local output="$2"
  magick "$MASTER" \
    -resize "${size}x${size}" \
    -alpha off \
    -depth 8 \
    -strip \
    -define png:color-type=2 \
    "$output"
}

magick "$SOURCE" \
  -resize 1024x1024! \
  -alpha off \
  -depth 8 \
  -strip \
  -define png:color-type=2 \
  "$MASTER"

MAC="apps/dudka/macos/Runner/Assets.xcassets/AppIcon.appiconset"
for size in 16 32 64 128 256 512 1024; do
  render "$size" "$MAC/app_icon_${size}.png"
done

IOS="apps/dudka/ios/Runner/Assets.xcassets/AppIcon.appiconset"
while read -r filename size; do
  render "$size" "$IOS/$filename"
done <<'EOF'
Icon-App-20x20@1x.png 20
Icon-App-20x20@2x.png 40
Icon-App-20x20@3x.png 60
Icon-App-29x29@1x.png 29
Icon-App-29x29@2x.png 58
Icon-App-29x29@3x.png 87
Icon-App-40x40@1x.png 40
Icon-App-40x40@2x.png 80
Icon-App-40x40@3x.png 120
Icon-App-60x60@2x.png 120
Icon-App-60x60@3x.png 180
Icon-App-76x76@1x.png 76
Icon-App-76x76@2x.png 152
Icon-App-83.5x83.5@2x.png 167
Icon-App-1024x1024@1x.png 1024
EOF

ANDROID="apps/dudka/android/app/src/main/res"
render 48 "$ANDROID/mipmap-mdpi/ic_launcher.png"
render 72 "$ANDROID/mipmap-hdpi/ic_launcher.png"
render 96 "$ANDROID/mipmap-xhdpi/ic_launcher.png"
render 144 "$ANDROID/mipmap-xxhdpi/ic_launcher.png"
render 192 "$ANDROID/mipmap-xxxhdpi/ic_launcher.png"

magick "$MASTER" \
  -define icon:auto-resize=256,128,64,48,32,16 \
  "apps/dudka/windows/runner/resources/app_icon.ico"

echo "generate_app_icons OK master=$MASTER"
