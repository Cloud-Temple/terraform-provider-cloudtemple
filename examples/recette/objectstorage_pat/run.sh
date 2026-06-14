#!/usr/bin/env bash
#
# Phase 1 recette wrapper — PAT + object storage (SUPERVISED, human-run only).
#
# This script is INERT in the harness: it is only run by a human under explicit
# GO during the supervised phase (Phase B of the #300 design). It builds the
# Phase 1 stack, lets the operator inspect it, and ALWAYS tears it down.
#
# SAFETY (non-negotiable, SecNumCloud posture):
#   - NO `set -x`: command tracing would echo resolved secrets/role names.
#   - NO env echo / no `env` / no `printenv`: the environment carries the PAT
#     credentials and the storage secret.
#   - NO TF_LOG=DEBUG: the provider transport masks the PAT body but NOT the
#     object-storage secretAccessKey, so DEBUG logs can leak the storage secret.
#   - Fail-closed tenant guard BEFORE apply: a non-mutating Go pre-flight
#     authenticates and asserts the tenant equals CLOUDTEMPLE_RECETTE_TENANT_ID.
#     The pre-flight runs ONLY the guard (TestRecetteLiveGuardOnly); the recette
#     TestMain has no auto-cleanup, so this pre-flight creates and destroys
#     nothing — the "no mutation" label below is literally true.
#   - trap on EXIT runs `terraform destroy` on success, failure, or Ctrl-C.
#
# This script NEVER sweeps. If you want a clean slate before the apply, run the
# explicit, DESTRUCTIVE sweep yourself, on purpose, BEFORE invoking this script:
#
#     TF_ACC=1 go test ./internal/provider/tests/recette/... \
#       -sweep=recette -sweep-allow-failures
#
# That -sweep deletes recette-scoped objects (it re-asserts the tenant guard
# first). It is intentionally NOT auto-run here: a destructive delete must be an
# explicit operator decision, never a hidden side effect of a guarded pre-flight.
#
# Credentials and the tenant allowlist are injected via the environment (sourced
# from a gitignored .env.recette or CI environment secrets). This script never
# reads .env.recette itself and never prints any value.

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "${SCRIPT_DIR}/../../.." && pwd)"

# --- Required environment (names only; values never printed) ----------------
: "${CLOUDTEMPLE_RECETTE_TENANT_ID:?CLOUDTEMPLE_RECETTE_TENANT_ID must be set (recette tenant allowlist)}"
: "${CLOUDTEMPLE_CLIENT_ID:?CLOUDTEMPLE_CLIENT_ID must be set}"
: "${CLOUDTEMPLE_SECRET_ID:?CLOUDTEMPLE_SECRET_ID must be set}"
: "${TF_VAR_iam_role_name:?TF_VAR_iam_role_name must be set (pre-existing IAM role name)}"
: "${TF_VAR_object_storage_role_name:?TF_VAR_object_storage_role_name must be set (pre-existing object-storage role name)}"

# --- Fail-closed tenant guard pre-flight (mutates nothing) ------------------
# Reuses the exact same guard as the Go harness. Aborts here if the authenticated
# tenant does not match the allowlist, BEFORE any terraform apply. The recette
# TestMain runs ONLY the guard (no auto-sweep), so this pre-flight truly mutates
# nothing: it creates and destroys no resource.
echo "Running tenant guard pre-flight (no mutation)..."
TF_ACC=1 go test "${REPO_ROOT}/internal/provider/tests/recette/" \
  -run '^TestRecetteLiveGuardOnly$' -count=1 -v

# --- Always destroy on exit -------------------------------------------------
cd "${SCRIPT_DIR}"
trap 'echo "Tearing down recette stack..."; terraform destroy -auto-approve' EXIT

terraform init -input=false
terraform validate

# Convergence: an apply, then a plan that must be empty (exit code 0).
terraform apply -auto-approve -input=false
echo "Stack built. Verifying convergence (empty second plan)..."
terraform plan -detailed-exitcode -input=false

echo "Recette stack is up. Inspect it now, then this script will destroy on exit."
# The trap above destroys when the script exits (normally or on interrupt).
