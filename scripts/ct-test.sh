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
machine-managers|machine_managers|0|1|OpenIaaS: just run_identity + machine_managers.list, repeatable, to characterize the #315 5xx flakiness. Read-only.
probe-absence|probe_absence|0|1|GET every by-id read with a bogus id to map the 404-vs-403 absence contract per endpoint (#384). Read-only; needs a read-entitled token (a 403 is "not migrated" only if the token may read the type).
vpc|vpc|1|0|QUARANTINED: /vpc/v1 is deprecated and frozen pending the rebuild (no cloudtemple_vpc_* provider surface ships in v1.8.0). The opt-in `ct-validate -cycles vpc -write` still runs it manually.
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

# Teardown state for the TF lifecycle (globals, because a RETURN/EXIT/signal trap
# fires after cmd_tf's locals are gone). _tf_teardown is the never-orphan safety net:
# if the normal flow did not reach its own destroy (interrupt / crash), it destroys
# the run-unique stack best-effort; then it always removes the isolated workdir.
_TF_WORK=""
_TF_RCFILE=""
_TF_RUN_NAME=""
_tf_teardown() {
  set +e            # never let the safety net abort half-way under `set -e`
  trap '' INT TERM  # a second Ctrl-C must not interrupt the net destroy
  [ -n "$_TF_WORK" ] && [ -d "$_TF_WORK" ] || return 0
  # The normal flow marks `.destroyed` only after a SUCCESSFUL destroy, so the net
  # re-attempts when the normal destroy failed or never ran (interrupt / crash).
  if [ ! -f "$_TF_WORK/.destroyed" ]; then
    echo ">> teardown net: destroying (normal destroy did not complete)..." >&2
    ( cd "$_TF_WORK" && TF_CLI_CONFIG_FILE="$_TF_RCFILE" terraform destroy -auto-approve -input=false -var "vm_name=$_TF_RUN_NAME" >&2 )
  fi
  rm -rf "$_TF_WORK"
}

cmd_tf() {
  local scenario="${1:-}"; [ -n "$scenario" ] || die "usage: ct-test tf <scenario> (see 'ct-test list')"
  local row; row="$(lookup "$scenario")" || true
  [ -n "$row" ] || die "unknown scenario '$scenario' (see 'ct-test list')"
  local dir="$REPO_ROOT/examples/ct-test/$scenario"
  [ -d "$dir" ] || die "scenario '$scenario' (tf) is not ready yet — no example at examples/ct-test/$scenario"
  command -v terraform >/dev/null || die "terraform not found in PATH"
  load_creds

  # Use the LOCALLY built provider (this checkout), not a registry release, via a
  # dev_overrides CLI config. With dev_overrides, `terraform init` is unnecessary
  # (and warns), so we skip it.
  echo ">> Building the provider (local dev_override)..."
  ( cd "$REPO_ROOT" && go build -o terraform-provider-cloudtemple . ) || die "provider build failed"

  # Run in an ISOLATED workdir so no terraform.tfstate ever persists in the repo: a
  # prior failed run can never contaminate this one (the run-unique name guards the
  # remote object; the isolated state guards the local state).
  local work; work="$(mktemp -d "${TMPDIR:-/tmp}/ct-test-tf.XXXXXX")"
  # Run-unique name so a prior run's orphan can never collide with this apply.
  local run_name="ct-validate-tf-$(date +%Y%m%d-%H%M%S)-$$"
  local rcfile="$work/dev.tfrc"
  # Arm the never-orphan net IMMEDIATELY (before any setup can fail), so the workdir
  # is always cleaned and any created stack is always swept.
  _TF_WORK="$work"; _TF_RCFILE="$rcfile"; _TF_RUN_NAME="$run_name"
  trap '_tf_teardown' EXIT
  trap 'echo ">> interrupted — tearing down..." >&2; exit 130' INT TERM

  cp "$dir"/*.tf "$work"/
  cat > "$rcfile" <<EOF
provider_installation {
  dev_overrides {
    "registry.terraform.io/Cloud-Temple/cloudtemple" = "$REPO_ROOT"
  }
  direct {}
}
EOF
  export TF_CLI_CONFIG_FILE="$rcfile"

  # Scenarios that declare a vm_power_state variable get a stop/start power cycle.
  local has_power=0
  grep -rqs 'variable "vm_power_state"' "$dir"/*.tf && has_power=1

  echo ">> TF scenario '$scenario'  name=$run_name  workdir=$work  (provider: local dev_override)"
  ( cd "$work"
    set +e

    terraform validate >/dev/null || { echo ">> FAIL: terraform validate" >&2; exit 1; }

    # apply_and_converge <label> [power]: apply, then assert `plan` is EMPTY
    # (-detailed-exitcode: 0=convergent, 2=drift, other=error). The empty-plan check
    # is the value of the TF path over the raw API: it proves no permanent drift
    # after apply. plan uses the SAME -var set as the apply (so flipping power_state
    # is not mistaken for drift).
    apply_and_converge() {
      local label="$1" power="${2:-}"
      echo ">> [$label] apply..."
      if [ -n "$power" ]; then
        terraform apply -auto-approve -input=false -var "vm_name=$run_name" -var "vm_power_state=$power" || { echo ">> [$label] FAIL: apply" >&2; return 1; }
      else
        terraform apply -auto-approve -input=false -var "vm_name=$run_name" || { echo ">> [$label] FAIL: apply" >&2; return 1; }
      fi
      echo ">> [$label] convergence (plan must be empty)..."
      # Capture the plan so a non-empty/errored plan can be SHOWN — a bare "drift
      # detected" with no diff is not actionable.
      local planout; planout="$(mktemp)"
      if [ -n "$power" ]; then
        terraform plan -detailed-exitcode -no-color -input=false -var "vm_name=$run_name" -var "vm_power_state=$power" >"$planout" 2>&1
      else
        terraform plan -detailed-exitcode -no-color -input=false -var "vm_name=$run_name" >"$planout" 2>&1
      fi
      local plan_rc=$?
      case $plan_rc in
        0) echo ">> [$label] OK — convergent (no drift)"; rm -f "$planout"; return 0 ;;
        2) echo ">> [$label] FAIL — drift detected (plan not empty); the drifting plan:" >&2
           sed 's/^/   | /' "$planout" >&2; rm -f "$planout"; return 1 ;;
        *) echo ">> [$label] FAIL — plan errored:" >&2
           sed 's/^/   | /' "$planout" >&2; rm -f "$planout"; return 1 ;;
      esac
    }

    lifecycle_fail=0
    if [ "$has_power" -eq 1 ]; then
      # create+start → convergence → stop → convergence → restart → convergence
      apply_and_converge "create+start" on || lifecycle_fail=1
      [ "$lifecycle_fail" -eq 0 ] && { apply_and_converge "stop" off || lifecycle_fail=1; }
      [ "$lifecycle_fail" -eq 0 ] && { apply_and_converge "restart" on || lifecycle_fail=1; }
    else
      apply_and_converge "create" || lifecycle_fail=1
    fi

    # Destroy ALWAYS — even on a lifecycle failure above — to never leave orphans.
    echo ">> Destroying (always, even on failure)..."
    terraform destroy -auto-approve -input=false -var "vm_name=$run_name"; destroy_rc=$?
    # Mark `.destroyed` ONLY on success, so the EXIT net re-attempts a failed destroy.
    [ "$destroy_rc" -eq 0 ] && : > .destroyed

    # Post-destroy orphan SMOKE CHECK — a best-effort net ON TOP of the destroy (which
    # is the PRIMARY proof of removal, via the provider's delete+wait). It scans the
    # live OpenIaaS listings for our run-unique name on the NAMED resources the
    # scenario creates as their own objects (the VM and the data disk). Semantics are
    # deliberately one-directional: it can only FAIL the run (a run-named object still
    # present == orphan); it NEVER upgrades a run to "proven clean". A listing is
    # consulted only when it is a trusted complete response (curl ok + HTTP 200 + body
    # is a JSON array); anything else is skipped (inconclusive), never read as absent.
    # A rigorous independent proof — complete ListStrict by id, incl. the VM's
    # sub-objects (OS disk, network adapter, which the VM delete is expected to
    # cascade) — is a follow-up; grep-by-name only covers the run-named resources.
    orphan_found=0; smoke_skipped=0
    echo ">> Post-destroy orphan smoke check (run-named VM/disk must be gone)..."
    local tok
    tok="$(curl -ksS --max-time 30 -X POST "https://${CT_TEST_HOST}/api/iam/v2/auth/personal_access_token" \
            -A "" -H 'Accept:' -H 'Content-Type: application/json' \
            -d "{\"id\":\"${CLOUDTEMPLE_CLIENT_ID}\",\"secret\":\"${CLOUDTEMPLE_SECRET_ID}\"}" 2>/dev/null)"
    # list_probe <path> <exact-json-name> → "present" | "absent" | "unavailable".
    # The pattern is the EXACT quoted JSON name, so the VM name does not falsely match
    # the "<name>-data" disk and vice versa.
    list_probe() {
      local path="$1" pat="$2" bf="$work/.list.json" code crc
      code="$(curl -ksS --max-time 30 -o "$bf" -w '%{http_code}' -A "" -H 'Accept:' \
               -H "Authorization: Bearer $tok" "https://${CT_TEST_HOST}${path}" 2>/dev/null)"; crc=$?
      if [ "$crc" -ne 0 ] || [ "$code" != "200" ] || [ "$(head -c1 "$bf" 2>/dev/null)" != "[" ]; then
        echo unavailable; return
      fi
      grep -qF "$pat" "$bf" && { echo present; return; } # -F: exact string, not a regex
      echo absent
    }
    # probe_one <path> <label> <exact-json-name>: FAIL on present, note skipped on inconclusive.
    probe_one() {
      local res; res="$(list_probe "$1" "$3")"
      case "$res" in
        present) echo ">>   FAIL — $2 orphan ($3) STILL PRESENT in $1" >&2; orphan_found=1 ;;
        absent)  echo ">>   smoke: no $2 orphan in $1" ;;
        *)       echo ">>   smoke: $1 inconclusive (skipped — not proof of absence)" >&2; smoke_skipped=1 ;;
      esac
    }
    case "$tok" in
      eyJ*)
        probe_one /api/compute/v1/open_iaas/virtual_machines "VM" "\"$run_name\""
        probe_one /api/compute/v1/open_iaas/virtual_disks "data-disk" "\"${run_name}-data\"" ;;
      *) echo ">>   smoke check skipped (could not authenticate)" >&2; smoke_skipped=1 ;;
    esac

    if [ "$lifecycle_fail" -ne 0 ] || [ "$destroy_rc" -ne 0 ] || [ "$orphan_found" -ne 0 ]; then
      echo ">> FAIL (lifecycle=$lifecycle_fail destroy=$destroy_rc orphan=$orphan_found)$([ "$destroy_rc" -ne 0 ] && echo ' — DESTROY FAILED, CHECK FOR ORPHANS')" >&2
      exit 1
    fi
    echo ">> OK (apply + convergence$([ "$has_power" -eq 1 ] && echo ' + stop/start cycle') + destroy clean; no orphan found in trusted listings$([ "$smoke_skipped" -ne 0 ] && echo ' — some smoke probes inconclusive'))"
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
