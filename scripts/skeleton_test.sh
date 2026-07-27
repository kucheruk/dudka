#!/usr/bin/env bash
# Task-level contract for P005: cmd/internal skeleton + version stubs.
# Run: ./scripts/skeleton_test.sh
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"
export DUDKA_NO_PROMPT=1

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

# dudkad stays up after ready; capture log then stop.
log_d="$tmpdir/dudkad.log"
"$tmpdir/dudkad" -data-dir "$tmpdir/data" -name "Skeleton" -listen "127.0.0.1:0" >"$log_d" 2>&1 &
pid=$!
for _ in $(seq 1 50); do
  grep -q '^ready ' "$log_d" 2>/dev/null && break
  kill -0 "$pid" 2>/dev/null || fail "dudkad exited early; log: $(cat "$log_d")"
  sleep 0.1
done
kill "$pid" 2>/dev/null || true
wait "$pid" 2>/dev/null || true

full_t="$("$tmpdir/dudka")"
out_d="$(head -n 1 "$log_d")"
out_t="$(printf '%s\n' "$full_t" | head -n 1)"

[[ "$out_d" == dudkad\ * ]] || fail "dudkad stdout want 'dudkad <version>', got: $out_d"
[[ "$out_t" == dudka\ * ]] || fail "dudka stdout want 'dudka <version>', got: $out_t"

ver_d="${out_d#dudkad }"
ver_t="${out_t#dudka }"
[[ -n "$ver_d" ]] || fail "dudkad printed empty version"
[[ "$ver_d" == "$ver_t" ]] || fail "version mismatch: dudkad=$ver_d dudka=$ver_t"

go test ./... >/dev/null || fail "go test ./... failed"

echo "skeleton_test OK"
