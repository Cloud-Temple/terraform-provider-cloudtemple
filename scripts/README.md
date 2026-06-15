# `scripts/`

Helper scripts for developing, testing and validating the Cloud Temple Terraform
provider. None of them are required to *use* the published provider — they support
contributors and CI.

| Script | Purpose | Needs live API? |
| ------ | ------- | --------------- |
| [`ct-test.sh`](ct-test.sh) | Run a business scenario against the live API (via the `ct-validate` runner) **or** via Terraform. | Yes |
| [`ct-soak.sh`](ct-soak.sh) | Bounded, breaker-guarded soak probe of one read-only endpoint, to characterize intermittent platform flakiness. | Yes (read-only) |
| [`coverage_ratchet.sh`](coverage_ratchet.sh) | CI coverage gate for the pure (platform-independent) packages. | No |
| [`coverage_floor.txt`](coverage_floor.txt) | Data file: the per-package coverage floors enforced by the ratchet. | — |

---

## `ct-test.sh` — run a scenario against the API or via Terraform

Loads a tenant's credentials, targets the API host (`shiva.cloud-temple.com`), plays
a business scenario, prints PASS/FAIL, and cleans up after itself.

```bash
scripts/ct-test.sh list                              # list scenarios and which modes are ready
scripts/ct-test.sh api  <scenario> [flags...]        # play the scenario directly against the API
scripts/ct-test.sh tf   <scenario>                   # play the same scenario via Terraform (apply + destroy)
```

- **Tenant selection** (which credentials, hence which platform you exercise):
  `--tenant openiaas` (default, uses `$CT_ENV_OPENIAAS`) or `--tenant vmware`
  (uses `$CT_ENV_VMWARE`); or set `CT_TENANT`, or `CT_TEST_ENV=<file>` to point at a
  creds file directly. The API host is the same for every tenant — the tenant is
  determined by the credentials, not the URL.
- **Credentials file** is `KEY=value` with `CLOUDTEMPLE_CLIENT_ID` /
  `CLOUDTEMPLE_SECRET_ID`.
- **Load knobs** (api mode) are forwarded verbatim to the `ct-validate` runner, so
  you can drive bounded repetition and parallelism — the circuit breaker still guards
  the API:

  ```bash
  scripts/ct-test.sh --tenant vmware api vm-vmware -runs 20 -concurrency 4
  ```

Scenarios today: `readonly` (safe, read-only), `machine-managers` (read-only: just
`run_identity` + `machine_managers.list`, repeatable — see below), `vpc`, `storage`,
`vm` (OpenIaaS lifecycle), `vm-vmware` (VMware lifecycle). Write scenarios create then
destroy their resources, with a deferred never-orphan teardown net.

### `machine-managers` — isolated, client-identical probe of one flaky call (#315)

A read-only scenario that performs EXACTLY the first two steps of the `vm` cycle — the
local `run_identity` token, then `compute.openiaas.machine_managers.list` (the same
`MachineManager().List(ctx)` call) — and nothing else. It is the faithful way to
reproduce the intermittent `machine_managers` 5xx (#315) in isolation, through the real
provider client (unlike a `curl` probe, it is byte-identical to what the `vm` cycle
sends). Drive it with load knobs:

```bash
# repeat the exact call 100× in series (default breaker stops on distress)
scripts/ct-test.sh api machine-managers -runs 100 -concurrency 1

# characterize the full 5xx rate over all 100 calls (relax the breaker), or chase the burst correlation
scripts/ct-test.sh api machine-managers -runs 100 -concurrency 1 \
  -abort-failure-rate 1.0 -abort-consecutive 1000 -abort-window 1000
scripts/ct-test.sh api machine-managers -runs 100 -concurrency 8
```

The per-endpoint report then gives the OK% / 5xx rate of `machine_managers.list`.

## `ct-soak.sh` — characterize intermittent endpoint flakiness

Authenticates with a PAT, then fires a fixed quota of GET calls per **concurrency
step** (1 → 2 → 4 → 8 …), measuring HTTP code and latency for each call. After every
step it reports OK% / 5xx / 4xx / errors and latency percentiles, and **stops the
ladder early** if a step's 5xx rate crosses a threshold — the bounded-probe /
stop-at-first-distress doctrine (never stress a shared API to breakage).

Read-only (GETs only — no writes, no orphans). Credentials and the bearer token are
never printed.

```bash
# default: OpenIaaS machine_managers list, steps "1 2 4 8", 25 calls/step (= 100 total)
CT_ENV=/path/to/.env.recette-openiaas scripts/ct-soak.sh

# more aggressive ladder, or probe a different read-only GET:
STEPS="1 2 4 8 16" PER_STEP=20 scripts/ct-soak.sh
API_PATH=/api/compute/v1/open_iaas/storage_repositories scripts/ct-soak.sh
```

| Env knob | Default | Meaning |
| -------- | ------- | ------- |
| `CT_ENV` | `$CT_ENV_OPENIAAS`, else `./.env.recette-openiaas` | creds file (`CLOUDTEMPLE_CLIENT_ID` / `CLOUDTEMPLE_SECRET_ID`) |
| `HOST` | `shiva.cloud-temple.com` | API host |
| `API_PATH` | `/api/compute/v1/open_iaas` | the GET path to probe (machine_managers list) |
| `STEPS` | `1 2 4 8` | concurrency ladder |
| `PER_STEP` | `25` | calls per step (total = `PER_STEP` × #steps) |
| `MAX_TIME` | `30` | per-call timeout (s) |
| `ABORT_5XX_RATE` | `0.5` | stop the ladder if a step's 5xx rate exceeds this |
| `USER_AGENT` | *(empty)* | `User-Agent` header (empty = byte-for-byte like the Go client) |

Exit code is non-zero if the breaker tripped. Useful to characterize platform
flakiness (e.g. ComputeManager 5xx bursts) and to size a client retry/backoff policy.

**Fidelity to the real client:** the probe sends the **same bytes** as the
`ct-validate` Go client — an empty `User-Agent` and no `Accept` header (curl would
otherwise inject `curl/x.y` + `Accept: */*`, which a gateway can route differently).
Even so, curl can never be 100% identical (HTTP version, TLS, header order, keep-alive).
**For a byte-identical soak, prefer the `machine-managers` scenario** (above), which
drives `machine_managers.list` through the real Go client:

```bash
CT_ENV_OPENIAAS=/path/.env.recette-openiaas scripts/ct-test.sh api machine-managers -runs 100 -concurrency 8
```

Use `ct-soak.sh` as a quick standalone curl probe; use `ct-test.sh api machine-managers`
(or `api readonly` for a multi-endpoint sweep) when you need the exact provider client.

## `coverage_ratchet.sh` — CI coverage gate

Runs the pure-package unit tests under the race detector with coverage and **fails if
any package's statement coverage drops below its recorded floor** in
[`coverage_floor.txt`](coverage_floor.txt). The floor is raised as the suite grows and
must never be lowered by hand — making "add tests where they are missing" a systemic CI
guard rather than a discretionary habit.

```bash
scripts/coverage_ratchet.sh            # run + gate (used by CI)
scripts/coverage_ratchet.sh --update   # run + rewrite the floor to current, then commit it
```

Portable bash 3.2+ (macOS and CI behave the same). No live platform required — only the
pure packages (`internal/client`, `internal/provider`, `internal/provider/helpers`) run.
