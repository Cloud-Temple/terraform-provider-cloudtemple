# Runbook — relaunch the TF VM test (`ct-test.sh tf vm`)

Quick relaunch checklist, caveats and troubleshooting for the OpenIaaS VM lifecycle
scenario. For *what the scenario exercises*, see [README.md](README.md); this file is
the operational "how to re-run it and what to watch for".

## Run

```bash
cd <repo-root>
CT_ENV_OPENIAAS=./.env.recette-openiaas scripts/ct-test.sh tf vm
```

- `tf` runs the scenario through real Terraform (apply + destroy) using **our** locally
  built provider via a `dev_overrides` CLI config (no `terraform init` needed).
- `CT_ENV_OPENIAAS` is overridden to `./.env.recette-openiaas` because the script's
  default points elsewhere. The file is `KEY=value`
  (`CLOUDTEMPLE_CLIENT_ID` / `CLOUDTEMPLE_SECRET_ID`); the script loads it itself.

## Prerequisites

- `terraform` on `PATH`.
- `./.env.recette-openiaas` present (OpenIaaS tenant credentials).
- A network named **`LAN`** on the tenant, present on **both** layers — the OpenIaaS
  network (the adapter connects here) and the VPC private network (the static IP is
  allocated here). They share the name by convention (true on `FW_AGENTIC`). Otherwise
  set `var.lan_network_name`.
- **One free (unbound) public floating IP** on the VPC — the scenario binds one.
- The marketplace image `Ubuntu 24.04 LTS` available, else set `var.marketplace_name`.

## Lifecycle

```
create+start → plan(empty) → stop → plan(empty) → restart → plan(empty) → destroy → orphan smoke check
```

- A **non-empty plan** (drift) or a **failed destroy** (possible orphan) fails the run.
- `destroy` always runs, even on a mid-lifecycle failure.
- The post-destroy smoke check looks for the run-named **VM** and **data disk** only
  (200-only listings); a `>> OK (...)` final line is PASS, any `>> FAIL ...` is failure.

## Tunables

TF variables (defaults in parentheses), see [variables.tf](variables.tf):
`lan_network_name` (LAN) · `marketplace_name` (Ubuntu 24.04 LTS) · `cpu` (2) ·
`memory_gib` (4) · `data_disk_gib` (1) · `min_free_gib` (20). To override in `tf` mode,
edit the defaults or drop a `*.auto.tfvars` in this directory.

## ⚠️ Known side effect — each run leaks one `xoa` static IP (issue #359)

The scenario attaches an adapter to the **VPC-backed** LAN network. On attach, the
platform **auto-creates an `xoa`-source static IP** for the adapter's MAC (in addition
to the Terraform-managed `custom` one). On `destroy`, Terraform deletes the `custom`
static IP, but the **`xoa` one is NOT released** (platform-side GC gap — see #359), and
the smoke check does not look at it.

➡️ **Every `tf vm` run leaves one more orphaned `xoa` static IP on LAN.** This is *not* a
test failure, but it accumulates. Audit/purge it separately: list each private network's
static IPs and cross-check against live VMs; orphaned `xoa` IPs are not deletable via the
API (platform/console only).

## Troubleshooting

| Symptom | Likely cause | Action |
|---|---|---|
| `item ... not found` at apply | marketplace name mismatch | set `var.marketplace_name` |
| `Expected exactly one ... network named "LAN"` | LAN missing/duplicated on one layer | set `var.lan_network_name` |
| FIP bind fails | no free (unbound) public IP | release/provision a floating IP |
| `credentials not set` | creds file missing/incomplete | check `.env.recette-openiaas` (KEY=value, CLIENT_ID/SECRET_ID) |
| drift (non-empty plan) | a real provider convergence regression | inspect the printed plan — it is a bug to fix |
| destroy fails | possible orphan | check the tenant for the run-named VM/disk manually |
