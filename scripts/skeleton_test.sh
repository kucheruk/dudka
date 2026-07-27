#!/usr/bin/env bash
# Task-level contract for P005: cmd/internal skeleton + version stubs.
# Run: ./scripts/skeleton_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

fail() {
  echo "skeleton_test FAIL: $*" >&2
  exit 1
}

for p in cmd/dudkad cmd/dudka internal/version; do
  [[ -d "$p" ]] || fail "directory missing: $p"
done

[[ -f cmd/dudkad/main.go ]] || fail "cmd/dudkad/main.go missing"
[[ -f cmd/dudka/main.go ]] || fail "cmd/dudka/main.go missing"
[[ -f internal/version/version.go ]] || fail "internal/version/version.go missing"

go list ./... | grep -qx 'dudka/cmd/dudkad' || fail "go list missing dudka/cmd/dudkad"
go list ./... | grep -qx 'dudka/cmd/dudka' || fail "go list missing dudka/cmd/dudka"
go list ./... | grep -qx 'dudka/internal/version' || fail "go list missing dudka/internal/version"

tmpdir="$(mktemp -d)"
trap 'rm -rf "$tmpdir"' EXIT

go build -o "$tmpdir/dudkad" ./cmd/dudkad || fail "go build ./cmd/dudkad failed"
go build -o "$tmpdir/dudka" ./cmd/dudka || fail "go build ./cmd/dudka failed"

# Stubs may print extra lines (e.g. peer_id); version contract is the first line.
# Capture full stdout first so head does not SIGPIPE the binary (exit 141).
full_d="$("$tmpdir/dudkad" -data-dir "$tmpdir/data")"
full_t="$("$tmpdir/dudka")"
out_d="$(printf '%s\n' "$full_d" | head -n 1)"
out_t="$(printf '%s\n' "$full_t" | head -n 1)"

[[ "$out_d" == dudkad\ * ]] || fail "dudkad stdout want 'dudkad <version>', got: $out_d"
[[ "$out_t" == dudka\ * ]] || fail "dudka stdout want 'dudka <version>', got: $out_t"

ver_d="${out_d#dudkad }"
ver_t="${out_t#dudka }"
[[ -n "$ver_d" ]] || fail "dudkad printed empty version"
[[ "$ver_d" == "$ver_t" ]] || fail "version mismatch: dudkad=$ver_d dudka=$ver_t"

go test ./... >/dev/null || fail "go test ./... failed"

echo "skeleton_test OK"
