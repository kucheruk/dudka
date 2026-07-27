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

echo "OK"
echo "  $OUT/dudka-linux-${ARCH}"
echo "  $OUT/dudkad-linux-${ARCH}"
echo "  $OUT/dudka -> dudka-linux-${ARCH}"
echo "  $OUT/dudkad -> dudkad-linux-${ARCH}"
