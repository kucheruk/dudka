#!/usr/bin/env bash
# Release contract: Go, Flutter and CHANGELOG expose the same product version.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "release_contract_test FAIL: $*" >&2
  exit 1
}

flutter_release="$(sed -n 's/^version: \([0-9][0-9.]*\)+\([0-9][0-9]*\)$/\1 \2/p' apps/dudka/pubspec.yaml)"
[[ -n "$flutter_release" ]] || fail "apps/dudka/pubspec.yaml must contain version: X.Y.Z+BUILD"
read -r flutter_version flutter_build <<<"$flutter_release"
[[ "$flutter_build" -gt 0 ]] || fail "Flutter build number must be positive"

go_version="$(sed -n 's/^const Version = "\([^"]*\)"$/\1/p' internal/version/version.go)"
[[ "$go_version" == "$flutter_version" ]] \
  || fail "Go version $go_version != Flutter version $flutter_version"

grep -Fq "## [$flutter_version]" CHANGELOG.md \
  || fail "CHANGELOG.md has no section for $flutter_version"

echo "release_contract_test OK version=$flutter_version build=$flutter_build"
