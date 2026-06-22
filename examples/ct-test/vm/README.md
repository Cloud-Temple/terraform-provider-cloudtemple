# ct-test scenario `vm` — OpenIaaS VM full lifecycle (Terraform path)

Validates a full OpenIaaS VM lifecycle through the **real provider** (not the raw
API): deploy an Ubuntu VM **from the marketplace**, attach a data disk, then drive
power (start → stop → start), with a **convergence check** (empty `terraform plan`)
after every step and a clean `destroy` at the end.

## Run

```bash
CT_ENV_OPENIAAS=/path/to/.env.recette-openiaas scripts/ct-test.sh tf vm
```

`ct-test.sh` builds the provider from this checkout, wires it via a `dev_overrides`
CLI config (so Terraform uses *our* provider, not a registry release — no `init`
needed), injects a run-unique `vm_name`, and runs:

```
create+start → plan(empty) → stop → plan(empty) → restart → plan(empty) → destroy
```

A non-empty plan (drift) or a failed destroy (possible orphan) fails the run. The
destroy always runs, even on a mid-lifecycle failure, to never leave orphans.

## What it exercises

- **Marketplace deploy** (`marketplace_item_id` + `storage_repository_id`).
- **Data disk** attached to the VM (`cloudtemple_compute_iaas_opensource_virtual_disk`).
- **LAN**: the adapter joins the LAN, an OpenIaaS network selected by name.
- **Power management** via `power_state` (`on`/`off`) — the stop/start cycle.
- **Convergence**: no permanent drift after apply (the value of the TF path vs the API).
- **Clean teardown**: destroy leaves nothing behind.

`terraform output` exposes `vm_id`, `vm_power_state`, and `data_disk_id`.

## Substrate discovery (no hard-coded ids)

The availability zone and backup policy are the first the API returns; the
**storage repository** is the usable one (not in maintenance, accessible) with the
**most free capacity** — never the first listed, which may be full (a VM
precondition fails closed if none has room for the OS disk + data disk). The config
thus runs on any OpenIaaS tenant — mirroring the API cycle's read-then-pick
approach. A backup SLA policy is required to power a VM on; any policy in the tenant
satisfies it.

**The LAN** is selected **by name** (`var.lan_network_name`): the VM's adapter joins
the OpenIaaS network of that name. A resource `precondition` asserts the name matches
**exactly one** network (`length(...) == 1`), so a typo, a missing LAN, or an
ambiguous name is a clear plan-time error — never a wrong silent pick.

> Networking is the most tenant-specific part: it assumes an OpenIaaS network named
> `var.lan_network_name` exists. Validate on first live run and adjust if needed.

## Tunables (see `variables.tf`)

If an apply fails on a tenant-specific detail, adjust via `-var` or a `*.tfvars`:

| Variable | Default | When to change |
| -------- | ------- | -------------- |
| `lan_network_name` | `LAN` | Your LAN is named differently (matched on the OpenIaaS network the adapter joins). |
| `marketplace_name` | `Ubuntu 24.04 LTS` | The image name differs in your catalog (apply reports "not found"). |
| `cpu` / `memory_gib` | `2` / `4` | The image requires more (deploy rejected with `MEMORY_CONSTRAINT_VIOLATION_ORDER`). |
| `data_disk_gib` | `1` | Larger data disk. |
| `vm_power_state` | `on` | Driven by `ct-test.sh`; override only for a manual apply. |

Example: `cd examples/ct-test/vm && terraform apply -var marketplace_name="Ubuntu 22.04 LTS" -var memory_gib=8`
(but prefer `scripts/ct-test.sh tf vm`, which wires the local provider and the full lifecycle).
