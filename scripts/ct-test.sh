#!/usr/bin/env bash
#
# ct-test — run a business scenario against the API OR via Terraform.
#
#   ct-test list                          list the scenarios and which modes are ready
#   ct-test api  <scenario> [flags...]    play the scenario directly against the API
#   ct-test tf   <scenario>               play the same scenario via Terraform (apply + destroy)
#
# It loads the chosen tenant's credentials, targets the API host, runs the
# scenario, prints PASS/FAIL, and cleans up after itself.
#
# TENANT SELECTION (which credentials, hence which platform you exercise):
#   --tenant openiaas   (default) use $CT_ENV_OPENIAAS
#   --tenant vmware               use $CT_ENV_VMWARE
#   ... or set CT_TENANT=openiaas|vmware, or CT_TEST_ENV=<file> to point at one directly.
# The API host is the same for every tenant (shiva.cloud-temple.com); the tenant
# is determined by the credentials, not by the URL.
#
# LOAD KNOBS (api mode): any extra flags after the scenario are forwarded verbatim
# to the runner, so you can drive repetition and parallelism:
#   ct-test --tenant vmware api vm-vmware -runs 20 -concurrency 4
#   (-runs N, -concurrency M, -timeout D ; the circuit breaker still guards the API.)
#
# Overridable via env:
#   CT_ENV_OPENIAAS  creds file for the openiaas tenant (default: agentic cloudtemple.env)
#   CT_ENV_VMWARE    creds file for the vmware tenant   (no default — set it)
#   CT_TEST_ENV      explicit creds file (wins over --tenant; KEY=value format)
#   CT_TEST_HOST     API host (default: shiva.cloud-temple.com — the confirmed API endpoint)
set -euo pipefail

CT_ENV_OPENIAAS="${CT_ENV_OPENIAAS:-/Users/clesur/PROJETS/AGENTIC-PLATFORM/vault/rendered/cloudtemple.env}"
CT_ENV_VMWARE="${CT_ENV_VMWARE:-}"
CT_TENANT="${CT_TENANT:-openiaas}"
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
vm|compute_lifecycle|1|1|OpenIaaS: create a VM from a template, add a disk, connect the network, then destroy.
vm-vmware|compute_vmware_lifecycle|1|1|VMware: create a VM (datacenter/host/datastore/guest-OS), add a disk, connect a NIC, then destroy.
EOF
}

die() { echo "ct-test: $*" >&2; exit 2; }

# resolve_env_file echoes the creds file to load, honouring (in priority order)
# an explicit CT_TEST_ENV, then the selected tenant's file.
resolve_env_file() {
  if [ -n "${CT_TEST_ENV:-}" ]; then
    echo "$CT_TEST_ENV"; return
  fi
  case "$CT_TENANT" in
    openiaas) echo "$CT_ENV_OPENIAAS" ;;
    vmware)
      [ -n "$CT_ENV_VMWARE" ] || die "tenant 'vmware' selected but CT_ENV_VMWARE is not set — point it at a KEY=value creds file (CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID), or pass CT_TEST_ENV=<file>"
      echo "$CT_ENV_VMWARE" ;;
    *) die "unknown tenant '$CT_TENANT' (use openiaas|vmware)" ;;
  esac
}

load_creds() {
  local env_file; env_file="$(resolve_env_file)"
  [ -f "$env_file" ] || die "credentials file not found: $env_file"
  set -a; . "$env_file" 2>/dev/null; set +a
  export CLOUDTEMPLE_HTTP_ADDR="$CT_TEST_HOST"
  export CLOUDTEMPLE_HTTP_SCHEME="https"
  [ -n "${CLOUDTEMPLE_CLIENT_ID:-}" ] && [ -n "${CLOUDTEMPLE_SECRET_ID:-}" ] \
    || die "credentials not set in $env_file (need CLOUDTEMPLE_CLIENT_ID / CLOUDTEMPLE_SECRET_ID)"
}

lookup() { # $1=scenario -> echoes "cycle write ready desc" or empty
  scenarios | awk -F'|' -v s="$1" '$1==s {print $2"\t"$3"\t"$4"\t"$5}'
}

cmd_list() {
  echo "Scenarios (use: ct-test [--tenant openiaas|vmware] api <name>  |  ct-test tf <name>):"
  scenarios | awk -F'|' '{
    ready = ($4==1) ? "ready" : "coming";
    printf "  %-10s [%-6s] %s\n", $1, ready, $5
  }'
  echo
  echo "Tenant: ${CT_TENANT}   Host (API): ${CT_TEST_HOST}"
  echo "Creds : openiaas=${CT_ENV_OPENIAAS}"
  echo "        vmware=${CT_ENV_VMWARE:-<unset: set CT_ENV_VMWARE>}"
}

cmd_api() {
  local scenario="${1:-}"; shift || true
  [ -n "$scenario" ] || die "usage: ct-test api <scenario> [flags...] (see 'ct-test list')"
  local row; row="$(lookup "$scenario")" || true
  [ -n "$row" ] || die "unknown scenario '$scenario' (see 'ct-test list')"
  local cycle write ready
  cycle="$(echo "$row" | cut -f1)"; write="$(echo "$row" | cut -f2)"; ready="$(echo "$row" | cut -f3)"
  [ "$ready" = "1" ] || die "scenario '$scenario' (api) is not ready yet — being built"

  echo ">> Building the API runner..."
  ( cd "$REPO_ROOT" && go build -o "$CTV_BIN" ./cmd/ct-validate )
  load_creds

  # Conservative defaults; any extra flags passed after the scenario (e.g.
  # -runs 20 -concurrency 4) are forwarded verbatim and override these
  # (Go's flag package keeps the last occurrence).
  local args=( -cycles "$cycle" -runs 1 -concurrency 1 -timeout 10m )
  [ "$write" = "1" ] && args+=( -write )
  args+=( "$@" )
  echo ">> API scenario '$scenario'  (tenant=$CT_TENANT, cycle=$cycle, write=$write)  flags: ${*:-<defaults>}"
  "$CTV_BIN" "${args[@]}"
}

cmd_tf() {
  local scenario="${1:-}"; [ -n "$scenario" ] || die "usage: ct-test tf <scenario> (see 'ct-test list')"
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
  # Peel off the global --tenant/-t option wherever it appears before the command.
  local positional=()
  while [ $# -gt 0 ]; do
    case "$1" in
      --tenant|-t) [ -n "${2:-}" ] || die "--tenant needs a value (openiaas|vmware)"; CT_TENANT="$2"; shift 2 ;;
      --tenant=*)  CT_TENANT="${1#*=}"; shift ;;
      *) positional+=("$1"); shift ;;
    esac
  done
  set -- "${positional[@]}"

  local cmd="${1:-}"; shift || true
  case "$cmd" in
    list)        cmd_list ;;
    api)         cmd_api "$@" ;;
    tf)          cmd_tf  "${1:-}" ;;
    ""|-h|--help)
      echo "usage: ct-test [--tenant openiaas|vmware] {list | api <scenario> [flags...] | tf <scenario>}"
      echo "       ct-test list                                  # see scenarios"
      echo "       ct-test --tenant vmware api vm-vmware          # full VMware VM lifecycle"
      echo "       ct-test --tenant vmware api vm-vmware -runs 20 -concurrency 4   # bounded load"
      ;;
    *)           die "unknown command '$cmd' (use: list | api <scenario> | tf <scenario>)" ;;
  esac
}

main "$@"
