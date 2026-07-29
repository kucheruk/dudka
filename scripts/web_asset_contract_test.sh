#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

hash_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | cut -c1-12
  else
    shasum -a 256 "$1" | cut -c1-12
  fi
}

check_asset() {
  local asset="$1"
  local expected
  expected="$(hash_file "web/$asset")"
  grep -Fq "$asset?v=$expected" web/index.html || {
    echo "web_asset_contract_test FAIL: $asset должен иметь ?v=$expected" >&2
    exit 1
  }
}

check_asset app.js
check_asset app.css
check_asset icon.png
echo "web_asset_contract_test OK"
