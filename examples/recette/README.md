# Recette harness (live acceptance) — Phase 1

This directory holds the **supervised, human-run** artifacts of the live
acceptance ("recette") harness for issue **#300**. The full design lives in the
[#300 design comment](https://github.com/Cloud-Temple/terraform-provider-cloudtemple/issues/300)
— this README does not duplicate it; it documents only what Phase 1 ships and
how to run it safely.

> **Nothing here runs automatically.** The Go harness skips without `TF_ACC`, and
> the standalone stack is run by a human under explicit GO. No live execution is
> performed by Phase 1 code.

## Phase 1 scope

Exactly the resources buildable on a **truly empty** tenant with **no compute
substrate**:

- `cloudtemple_iam_personal_access_token` (needs a pre-existing IAM role)
- `cloudtemple_object_storage_bucket`
- `cloudtemple_object_storage_storage_account`
- `cloudtemple_object_storage_acl_entry` (bucket × object-storage role-by-name × account)

**Excluded:** all compute (deferred on the Q1 substrate question) and
`global_access_key` (cannot be deleted via the API).

### Substrate assumption

Even an "empty" recette tenant must already carry, pre-existing and
datasource-only: **one IAM role** (for the PAT — PAT create hard-fails with no
role), **one object-storage role** (for the ACL), the **object-storage
service/namespace enabled**, and credentials with **object-storage + activity
read/wait** permissions. These are injected by env-var name only (see
`.env.recette.dist`).

## Safety model (the point of this harness)

Tenant scoping is **credential-only** (the PAT's JWT `scope.id`). The whole
safety story is: *prove, before any mutation, that the credentials point at the
allowlisted recette tenant — and abort hard otherwise.*

- **Env-var allowlist.** `CLOUDTEMPLE_RECETTE_TENANT_ID` is a **required**
  runtime value. It is never committed and never printed. The guard refuses to
  fall back to "current tenant".
- **Fail-closed, un-skippable guard.** The recette sub-package
  (`internal/provider/tests/recette/`) has its **own** `TestMain` that, in live
  mode (`TF_ACC`) or sweep mode (`-sweep`), authenticates and asserts the tenant
  equals the allowlist **before any TestCase or sweeper runs**, aborting fatally
  on mismatch. Each TestCase `PreCheck` re-asserts the guard as defence in depth,
  but the real un-skippability is the `TestMain` gate.
- **Redacted errors.** A mismatch aborts with a generic message that names only
  the env var; it never prints the expected or the actual tenant UUID. No UUID is
  ever committed.
- **Default path is safe.** With `TF_ACC` unset and no `-sweep`, the guard is
  skipped, the pure unit tests run without credentials and without any network
  call, and the package never hard-exits.

### Leak bans (SecNumCloud posture)

- **No `TF_LOG=DEBUG` capture/artifacts.** The provider transport masks the PAT
  body and `Authorization`, but **not** the object-storage `secretAccessKey`, so
  `DEBUG` logs can leak the storage secret.
- **No `set -x`, no env echo** in any script (`run.sh` enforces this).
- **No tenant UUID in errors or commits.** The allowlist is the env var's runtime
  value only.

## Teardown layering

1. **SDK auto-destroy** per TestCase (primary), even on a failed step.
2. **Explicit `-sweep` clean slate / orphan cleanup** (documented below). There is
   **no** automatic start-of-run cleanup: the recette `TestMain` mutates nothing on
   its own (it only runs the tenant guard). A clean slate is obtained by running
   the explicit, destructive `-sweep` **before** a run — never automatically.
   Sweepers do **not** auto-run; they fire only with the `-sweep` flag, and each
   re-asserts the tenant guard before deleting anything. Deletes are scoped to the
   `test-terraform*` name prefix / `created_by=Terraform` tag; PAT deletes are
   additionally filtered to the principal's own `UserId` + `TenantId`.

## How to run (each live step needs explicit human GO)

Credentials and the allowlist are injected via the environment. Copy the
template and fill it (never commit it):

```sh
cp .env.recette.dist .env.recette   # .env.recette is gitignored
# edit .env.recette, then:
set -a; . ./.env.recette; set +a
```

### Phase A — compile-and-skip (no mutation)

```sh
# Builds, runs the guard unit tests, and SKIPS the live tests (no TF_ACC).
go test ./internal/provider/tests/recette/... -run 'Recette' -v
```

### Phase B — supervised build + destroy (explicit GO)

Either drive the Go lifecycle harness:

```sh
TF_ACC=1 go test ./internal/provider/tests/recette/... -run 'TestAccRecette' -v -count=1
```

…or run the standalone stack (it guards the tenant, builds, verifies an empty
second plan, lets you inspect, and **always destroys on exit**):

```sh
./examples/recette/objectstorage_pat/run.sh
```

### Explicit `-sweep` (clean slate before a run / orphan cleanup after)

This is the **only** automatic-mutation-free destructive step, and it is fully
**opt-in**: nothing sweeps unless you pass `-sweep`. Run it **before** a run to
get a clean slate, or **after** a run to clear orphans. It re-asserts the tenant
guard, then deletes only recette-scoped objects.

```sh
# DESTRUCTIVE — run on purpose, never automatic.
TF_ACC=1 go test ./internal/provider/tests/recette/... -sweep=recette -sweep-allow-failures
```

### Phase C — CI integration

Out of Phase 1 scope. CI activation must be gated behind a GitHub
`environment: recette` (required reviewers), `workflow_dispatch`/`schedule`
triggers, and an `if: always()` teardown step. No live trigger is enabled here.

## Known convergence risk (bucket/acl drift)

The bucket resource's `Read` re-populates its **inline** `acl_entry` (an
`Optional` field) from the live ACL listing. The Phase 1 stack manages the ACL
**only** via the standalone `acl_entry` resource and keeps the bucket config free
of an inline block. If, in live mode, the bucket `Read` writes a non-empty inline
`acl_entry` into the bucket state while the config declares none, the second plan
will not be empty — a genuine state-safety finding to report on #300, **not** to
paper over. The lifecycle test deliberately does not set `ExpectNonEmptyPlan`, so
this surfaces as a hard failure if it occurs.
