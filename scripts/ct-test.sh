#!/usr/bin/env bash
#
# ct-test — run a business scenario against the API OR via Terraform.
#
#   ct-test list                 list the scenarios and which modes are ready
#   ct-test api  <scenario>      play the scenario directly against the API
#   ct-test tf   <scenario>      play the same scenario via Terraform (apply + destroy)
#
# It loads the agentic credentials, targets the confirmed API host, runs the
# scenario, prints PASS/FAIL, and cleans up after itself. No flags to remember.
#
# Overridable via env:
#   CT_TEST_ENV    path to the credentials env file (CLOUDTEMPLE_CLIENT_ID/SECRET_ID)
#   CT_TEST_HOST   API host (default: shiva.cloud-temple.com — the confirmed API endpoint)
set -euo pipefail

CT_TEST_ENV="${CT_TEST_ENV:-/Users/clesur/PROJETS/AGENTIC-PLATFORM/vault/rendered/cloudtemple.env}"
CT_TEST_HOST="${CT_TEST_HOST:-shiva.cloud-temple.com}"

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CTV_BIN="${CTV_BIN:-/tmp/ct-validate-bin}"

# scenario -> api cycle name (ct-validate), and whether it writes.
# ready=1 means the scenario is wired and runnable today.
#   name|api_cycle|write|ready|description
scenarios() {
  cat <<'EOF'
readonly|readonly|0|1|Read every service (no create/destroy). The safest scenario, validates the tool.
vpc|vpc|1|1|Create a VPC static IP + floating-IP binding, verify, then destroy.
storage|object_storage|1|1|Create an object-storage bucket + account + ACL, verify, then destroy.
vm|compute_lifecycle|1|1|Create a VM from a template, add a disk, connect the network, then destroy.
EOF
}

die() { echo "ct-test: $*" >&2; exit 2; }

load_creds() {
  [ -f "$CT_TEST_ENV" ] || die "credentials file not found: $CT_TEST_ENV (set CT_TEST_ENV)"
  set -a; . "$CT_TEST_ENV" 2>/dev/null; set +a
  export CLOUDTEMPLE_HTTP_ADDR="$CT_TEST_HOST"
  export CLOUDTEMPLE_HTTP_SCHEME="https"
  [ -n "${CLOUDTEMPLE_CLIENT_ID:-}" ] && [ -n "${CLOUDTEMPLE_SECRET_ID:-}" ] \
    || die "credentials not set in $CT_TEST_ENV (CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID)"
}

lookup() { # $1=scenario -> echoes "cycle write ready desc" or empty
  scenarios | awk -F'|' -v s="$1" '$1==s {print $2"\t"$3"\t"$4"\t"$5}'
}

cmd_list() {
  echo "Scenarios (use: ct-test api <name>  |  ct-test tf <name>):"
  scenarios | awk -F'|' '{
    ready = ($4==1) ? "ready" : "coming";
    printf "  %-10s [%-6s] %s\n", $1, ready, $5
  }'
  echo
  echo "Host (API): ${CT_TEST_HOST}   Creds file: ${CT_TEST_ENV}"
}

cmd_api() {
  local scenario="$1"; [ -n "$scenario" ] || die "usage: ct-test api <scenario> (see 'ct-test list')"
  local row; row="$(lookup "$scenario")" || true
  [ -n "$row" ] || die "unknown scenario '$scenario' (see 'ct-test list')"
  local cycle write ready desc
  cycle="$(echo "$row" | cut -f1)"; write="$(echo "$row" | cut -f2)"; ready="$(echo "$row" | cut -f3)"
  [ "$ready" = "1" ] || die "scenario '$scenario' (api) is not ready yet — being built"

  echo ">> Building the API runner..."
  ( cd "$REPO_ROOT" && go build -o "$CTV_BIN" ./cmd/ct-validate )
  load_creds

  local args=( -cycles "$cycle" -runs 1 -concurrency 1 -timeout 5m )
  [ "$write" = "1" ] && args+=( -write )
  echo ">> API scenario '$scenario'  (cycle=$cycle, write=$write)"
  "$CTV_BIN" "${args[@]}"
}

cmd_tf() {
  local scenario="$1"; [ -n "$scenario" ] || die "usage: ct-test tf <scenario> (see 'ct-test list')"
  local row; row="$(lookup "$scenario")" || true
  [ -n "$row" ] || die "unknown scenario '$scenario' (see 'ct-test list')"
  local dir="$REPO_ROOT/examples/ct-test/$scenario"
  [ -d "$dir" ] || die "scenario '$scenario' (tf) is not ready yet — no example at examples/ct-test/$scenario"
  command -v terraform >/dev/null || die "terraform not found in PATH"
  load_creds

  echo ">> TF scenario '$scenario'  (apply then destroy)  dir=$dir"
  ( cd "$dir"
    terraform init -input=false >/dev/null
    set +e
    terraform apply -auto-approve -input=false; apply_rc=$?
    echo ">> Destroying (always, even on apply failure)..."
    terraform destroy -auto-approve -input=false; destroy_rc=$?
    # A failed destroy is a hard failure (possible orphan) — never report PASS
    # just because apply succeeded. Surface both codes.
    if [ "$apply_rc" -ne 0 ] || [ "$destroy_rc" -ne 0 ]; then
      echo ">> FAIL (apply=$apply_rc destroy=$destroy_rc)$([ "$destroy_rc" -ne 0 ] && echo ' — DESTROY FAILED, CHECK FOR ORPHANS')" >&2
      exit 1
    fi
    echo ">> OK (apply + destroy both clean)"
    exit 0
  )
}

main() {
  local cmd="${1:-}"; shift || true
  case "$cmd" in
    list)        cmd_list ;;
    api)         cmd_api "${1:-}" ;;
    tf)          cmd_tf  "${1:-}" ;;
    ""|-h|--help) echo "usage: ct-test {list | api <scenario> | tf <scenario>}"; echo "       ct-test list   # see scenarios" ;;
    *)           die "unknown command '$cmd' (use: list | api <scenario> | tf <scenario>)" ;;
  esac
}

main "$@"
