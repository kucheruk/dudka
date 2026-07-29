#!/bin/sh
set -eu

repo_dir=$(CDPATH= cd -- "$(dirname "$0")/.." && pwd)
dist_dir="${DUDKA_DIST_DIR:-$repo_dir/dist}"
mkdir -p "$dist_dir"

CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
  go build -trimpath -o "$dist_dir/dudka-signal-linux-amd64" ./cmd/dudka-signal

echo "OK $dist_dir/dudka-signal-linux-amd64"
