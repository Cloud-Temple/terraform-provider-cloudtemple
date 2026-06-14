#!/usr/bin/env bash
#
# Coverage ratchet for the platform-independent (pure) packages.
#
# Runs the pure-package unit tests under the race detector with coverage and
# FAILS if any package's statement coverage drops below its recorded floor in
# scripts/coverage_floor.txt. The floor is raised as the test suite grows
# (run with --update after coverage improves, then commit the new floor); it
# must never be lowered by hand. This makes "add tests where they are missing"
# a systemic CI guard rather than a discretionary habit (issue #293).
#
# A package listed below with NO recorded floor is a FAIL in gate mode (a
# missing/typo'd floor line must not silently disable the gate); use --update
# to bootstrap it deliberately.
#
# Usage:
#   scripts/coverage_ratchet.sh            # run + gate (used by CI)
#   scripts/coverage_ratchet.sh --update   # run + rewrite the floor to current
#
# Portable bash 3.2+ (no associative arrays): macOS and CI behave the same.
# No live Cloud Temple platform is required: only the pure packages are run.
set -euo pipefail

cd "$(dirname "$0")/.."

FLOOR_FILE="scripts/coverage_floor.txt"
PKGS="internal/client internal/provider internal/provider/helpers"
# Tolerance (percentage points) absorbing float-formatting noise only.
EPS="0.05"

UPDATE=0
[[ "${1:-}" == "--update" ]] && UPDATE=1

# Prints the recorded floor for a package, or nothing if absent.
floor_for() {
  [[ -f "$FLOOR_FILE" ]] || return 0
  awk -v p="$1" '$1==p {print $2}' "$FLOOR_FILE"
}

fail=0
tmp_floor="$(mktemp)"
{
  echo "# Per-package statement-coverage floor (percent) for the pure packages."
  echo "# Raised as coverage improves (scripts/coverage_ratchet.sh --update);"
  echo "# CI fails if coverage drops below these values. Never lower by hand."
} > "$tmp_floor"

for pkg in $PKGS; do
  prof="$(mktemp)"
  # -covermode=atomic is required together with -race.
  go test -race -covermode=atomic -coverprofile="$prof" -count=1 "./$pkg"
  pct="$(go tool cover -func="$prof" | tail -1 | awk '{print $NF}' | tr -d '%')"
  rm -f "$prof"
  printf '%s %s\n' "$pkg" "$pct" >> "$tmp_floor"

  floor="$(floor_for "$pkg")"
  if [[ -z "$floor" ]]; then
    if [[ "$UPDATE" -eq 1 ]]; then
      floor="0"
    else
      printf 'RATCHET FAIL  %-32s no recorded floor (bootstrap with --update)\n' "$pkg"
      fail=1
      continue
    fi
  fi

  if awk -v p="$pct" -v f="$floor" -v e="$EPS" 'BEGIN{exit !(p + e < f)}'; then
    printf 'RATCHET FAIL  %-32s %s%% < floor %s%%\n' "$pkg" "$pct" "$floor"
    fail=1
  else
    printf 'ratchet ok    %-32s %s%% (floor %s%%)\n' "$pkg" "$pct" "$floor"
  fi
done

if [[ "$UPDATE" -eq 1 ]]; then
  mv "$tmp_floor" "$FLOOR_FILE"
  echo "floor updated -> $FLOOR_FILE"
else
  rm -f "$tmp_floor"
fi

if [[ "$fail" -ne 0 ]]; then
  echo "Coverage dropped below the floor. Add tests or, if intentional, justify and re-baseline with --update." >&2
  exit 1
fi
