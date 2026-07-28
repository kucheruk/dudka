#!/usr/bin/env bash
# P080: one-command Linux TUI (+ engine) build → artifacts under dist/.
# Usage: ./scripts/build_linux_tui.sh
# Optional: GOARCH=arm64 DIST=/tmp/out ./scripts/build_linux_tui.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

ARCH="${GOARCH:-amd64}"
OUT="${DIST:-$ROOT/dist}"
mkdir -p "$OUT"
VERSION="$(sed -n 's/^const Version = "\(.*\)"/\1/p' internal/version/version.go)"

export CGO_ENABLED=0
export GOOS=linux
export GOARCH="$ARCH"

echo "building Linux TUI → $OUT/dudka-linux-${ARCH}"
go build -trimpath -ldflags='-s -w' -o "$OUT/dudka-linux-${ARCH}" ./cmd/dudka

echo "building Linux engine → $OUT/dudkad-linux-${ARCH}"
go build -trimpath -ldflags='-s -w' -o "$OUT/dudkad-linux-${ARCH}" ./cmd/dudkad

# Stable names for docs / copy-paste on a Linux box (same arch).
ln -sfn "dudka-linux-${ARCH}" "$OUT/dudka"
ln -sfn "dudkad-linux-${ARCH}" "$OUT/dudkad"

bundle_root="$(mktemp -d "${TMPDIR:-/tmp}/dudka-bundle.XXXXXX")"
trap 'rm -rf "$bundle_root"' EXIT
bundle="$bundle_root/dudka-linux-${ARCH}"
mkdir -p "$bundle"
cp "$OUT/dudka-linux-${ARCH}" "$bundle/dudka"
cp "$OUT/dudkad-linux-${ARCH}" "$bundle/dudkad"
tar -czf "$OUT/dudka-linux-${ARCH}.tar.gz" -C "$bundle_root" "dudka-linux-${ARCH}"

if [[ "$ARCH" == "amd64" ]]; then
  sha="$(sha256sum "$OUT/dudka-linux-amd64.tar.gz" | awk '{print $1}')"
  sed -e "s/@VERSION@/$VERSION/g" -e "s/@ARCHIVE_SHA256@/$sha/g" \
    packaging/linux/install.sh.in > "$OUT/install.sh"
  chmod +x "$OUT/install.sh"
fi

echo "OK"
echo "  $OUT/dudka-linux-${ARCH}"
echo "  $OUT/dudkad-linux-${ARCH}"
echo "  $OUT/dudka -> dudka-linux-${ARCH}"
echo "  $OUT/dudkad -> dudkad-linux-${ARCH}"
echo "  $OUT/dudka-linux-${ARCH}.tar.gz"
