#!/usr/bin/env bash
# The ONLY supported way to run the real-database tests, in CI and locally: same setup, same commands, same checker.
# Needs a throwaway Postgres reachable through PGHOST/PGPORT/PGUSER/PGPASSWORD (a superuser; the password is rotated).
set -uo pipefail
root="$(cd "$(dirname "$0")/../.." && pwd)"
tmp="$(mktemp -d)"; trap 'rm -rf "$tmp"' EXIT
bash "$root/scripts/ci/realdb-setup.sh" > "$tmp/env" || { echo "setup failed" >&2; exit 1; }
set -a; . "$tmp/env"; set +a
cd "$root/backend"
: > "$tmp/out.json"
while read -r kind dir pat; do
  [ "$kind" = package ] || continue
  go test -json -count=1 -p 1 -timeout 20m -run "$pat" -skip '^TestStructure_OnboardingTimingOnARealTemplate$' "./$dir/" >> "$tmp/out.json"
done < <(grep -v '^#' "$root/scripts/ci/realdb-packages.txt")
# The verdict comes from the checker, which also fails on a go test failure.
python3 "$root/scripts/ci/realdb-check.py" "$tmp/out.json" "$root/scripts/ci/realdb-packages.txt" "$root/backend"
