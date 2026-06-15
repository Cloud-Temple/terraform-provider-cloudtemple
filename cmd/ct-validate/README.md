# ct-validate — Cloud Temple API check & resilience tool

`ct-validate` checks that the Cloud Temple API behaves correctly and stays
responsive. It runs realistic business scenarios — read-only checks and,
on request, full create-and-clean-up lifecycles — directly against the API
(through the provider's `internal/client` library), and produces a clear
per-endpoint health report: success rate, latency (p50/p95), and a breakdown of
any errors.

It can also replay a scenario repeatedly and in parallel to observe how the API
holds up under increasing load.

## Safe by design

- **Read-only by default** — it only reads unless you explicitly ask for write
  scenarios.
- **Automatic back-off** — a built-in circuit breaker eases off and stops
  launching new work as soon as the API shows signs of strain, then reports
  where.
- **Cleans up after itself** — every resource a write scenario creates is
  automatically removed, even if a run is interrupted.
- **Sensible limits** — conservative defaults (low concurrency, a single run)
  and built-in ceilings keep a run bounded.

## Two ways to run it

| | What | When |
|---|---|---|
| `scripts/ct-test.sh` | A simple wrapper: loads your credentials, targets the API, no flags to remember. | Everyday use. |
| `go run ./cmd/ct-validate` (or a built binary) | The tool itself, with every option exposed. | Fine-grained control / CI. |

### Quick start (the wrapper)

```sh
scripts/ct-test.sh list            # show the available scenarios
scripts/ct-test.sh api readonly    # read every service and report its health
scripts/ct-test.sh api vpc         # VPC: create → verify → remove
scripts/ct-test.sh api storage     # object storage: create → verify → remove
scripts/ct-test.sh api vm          # full VM lifecycle: create → verify → remove
scripts/ct-test.sh tf  <scenario>  # run the same scenario through Terraform
```

`api` exercises the API directly. `tf` runs the scenario through Terraform
(`apply` then `destroy`), which additionally validates the provider end to end.

### Running the tool directly

```sh
go run ./cmd/ct-validate -list                    # list scenarios (no network)
go run ./cmd/ct-validate -cycles readonly -json   # a read-only health report
```

## Scenarios

| Scenario | Type | What it does |
|---|---|---|
| `readonly` | read | Lists every service and reads a sample item — the broad health map. |
| `backup` | read | Backup service reads. |
| `compute_openiaas` | read | OpenIaaS compute reads. |
| `vpc` | write | Allocate a VPC static IP and a floating-IP binding, verify, then remove. |
| `object_storage` | write | Create a bucket, a storage account and an ACL, verify, then remove. |
| `iam_pat` | write | Create a personal access token, verify, then remove. |
| `compute_lifecycle` | write | Create a VM from a template, attach a disk and a network adapter, then remove everything. |

Write scenarios run only when you pass `-write`; otherwise they are listed and
skipped.

## Options

| Option | Default | Meaning |
|---|---|---|
| `-cycles` | `readonly` | Comma-separated scenario names, or `all`. |
| `-write` | `false` | Enable the write scenarios (they create and remove resources). |
| `-runs` | `1` | How many times to repeat each scenario (up to 10000). |
| `-concurrency` | `2` | Number of parallel workers (up to 64). |
| `-timeout` | `30m` | Overall time limit for the run. |
| `-abort-consecutive` | `5` | Back off after this many consecutive failures. |
| `-abort-failure-rate` | `0.30` | Back off when the failure rate over the window reaches this. |
| `-abort-window` | `20` | Size of the rolling window for the rate above. |
| `-json` | `false` | Emit the report as JSON. |
| `-list` | `false` | List scenarios and exit (no network, no credentials). |
| `-api-suffix` | `true` | Prefix request paths with `/api`. |

## Credentials

Provide your Cloud Temple personal access token and the API host via environment
variables:

| Variable | Meaning |
|---|---|
| `CLOUDTEMPLE_CLIENT_ID` / `CLOUDTEMPLE_SECRET_ID` | Your personal access token. The tenant is determined by the token. |
| `CLOUDTEMPLE_HTTP_ADDR` | API host (e.g. `shiva.cloud-temple.com`). Use the API hostname, not your web-console URL. |
| `CLOUDTEMPLE_HTTP_SCHEME` | `https`. |

Before sending anything, the tool prints the resolved target so you can confirm
it, requires HTTPS, and checks that your credentials are set.

## The report

For each endpoint the report shows the success rate, latency (p50/p95) and a
breakdown of outcomes. A short **"where it squeaks"** section lists the
endpoints that did not return 100% success, worst first.

How to read it:

- A **5xx / timeout** points to a real server-side issue worth reporting.
- A **4xx** is a client-side result — most often a **service this tenant does
  not use** (for example VMware on an OpenIaaS-only tenant), which is expected.

Exit code: `0` if every endpoint succeeded, `1` if anything failed or the tool
backed off, `2` for a configuration error.

## Observing behaviour under load

To see how the API behaves as load increases, replay a scenario and raise the
load gradually between runs:

```sh
… -cycles compute_lifecycle -write -runs 20 -concurrency 2
… -cycles compute_lifecycle -write -runs 20 -concurrency 4
… -cycles compute_lifecycle -write -runs 20 -concurrency 8
```

The tool eases off automatically if the API starts to strain and reports exactly
where, so you can find the comfortable operating range without overloading the
service. For deliberate stress testing, point it at a dedicated environment.
