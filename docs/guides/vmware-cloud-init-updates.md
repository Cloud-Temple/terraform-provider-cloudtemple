# Updating VMware cloud-init

`cloudtemple_compute_virtual_machine.cloud_init` retains its existing map-of-strings
schema. Creation still passes these values to the Marketplace deployment options.
An update instead changes the VM's vApp properties using the VMware property API;
it does not replace the VM. Existing configurations with an unchanged `cloud_init`
map do not enter the new update path. There is no schema version change, state
migration, or new refresh mapping for this attribute.

Previously, Terraform could plan an in-place change and successfully apply it
without sending any cloud-init update. The state then contained the requested
values while the VM retained its original properties, and the next plan was empty.
The update now confirms the properties before recording success.

## Applying a change

The update always runs with the virtual machine stopped. The provider can take
care of that for you, or you can keep doing it in separate applies.

### Letting the provider power-cycle the machine

Set `allow_vm_restart = true` and apply the `cloud_init` change on its own:

```hcl
resource "cloudtemple_compute_virtual_machine" "vm" {
  # ...
  power_state      = "on"
  allow_vm_restart = true

  cloud_init = {
    hostname = "web-01"
  }
}
```

The provider then powers the machine off, edits and verifies the vApp
properties, and powers it back on. `allow_vm_restart` defaults to `false`:
without it, changing `cloud_init` on a running machine is refused at **plan**
time, so nothing is applied and nothing is stopped.

Two cases need no opt-in, because neither interrupts anything the operator did
not already ask to interrupt:

- `power_state = "off"` in the same apply — the machine is being stopped anyway,
  so the change is applied while it goes down and it is left stopped.
- a machine that is already stopped and is configured to end up running — it is
  started after the update. `allow_vm_restart` authorises *interrupting* a
  running machine, which is a different question.

Only `power_state` and `allow_vm_restart` may change in the same apply as
`cloud_init` in an **in-place update**. Every other change to the resource is
still refused there; apply it separately. A change to a `ForceNew` attribute
such as `deploy_options` is a different thing entirely: it asks for a
replacement, the new machine receives `cloud_init` through its own deployment,
and none of this update path — including the restart opt-in — applies to it. The configuration is validated in full **before** anything is
powered off, so a change that could never be applied costs no downtime.

A change that turns out to need no vApp write at all — because the machine
already holds the declared values — never interrupts anything: the provider
skips the power cycle entirely and only settles the declared power state. The
plan-time refusal cannot know this in advance, since it has no live evidence, so
it still asks for the opt-in first.

### Doing it in separate applies

1. Set `power_state = "off"` and apply.
2. Change only `cloud_init` and apply.
3. If required, update `os_network_adapter.ip_address` separately and apply while
   stopped. The guest network configuration and a VPC static IP reservation are
   separate settings; changing cloud-init alone does not change the reservation.
4. Set `power_state = "on"` and apply after the property update has succeeded.

### What the restart does, and what it does not do

The power cycle exists to make the vApp edit **safe**, not to make cloud-init run
again inside the guest. The two must not be confused:

- cloud-init distinguishes per-instance from per-boot modules. For a given
  instance, rebooting ordinarily does **not** re-run the per-instance
  configuration, so a changed `user-data` will usually not take effect in the
  guest just because the provider restarted the machine. See
  [cloud-init's first boot documentation](https://docs.cloud-init.io/en/latest/explanation/first_boot.html).
- the provider does not execute guest commands and never generates a new
  instance ID. For Ubuntu OVF images the declared `instance-id` is what
  distinguishes cloud-init instances; changing it can cause first-boot modules
  to run again, including modules unrelated to hostname or networking. Choose
  that value explicitly and verify your image's behaviour. See
  [Ubuntu's VMware configuration documentation](https://documentation.ubuntu.com/vmware/vmware-how-to/configure-your-vm/).

A successful apply means the vApp properties now hold the declared values and
the machine reached the declared power state. It does not mean the guest has
consumed them.

## API constraints

The public API provides add and delete operations, not an atomic edit. A
replacement first adds a property with the same textual ID and preserved metadata,
verifies that exactly that property was added, then deletes the original by its
numeric key. Duplicate IDs exist temporarily while the VM is stopped. All other
properties must remain unchanged in every verified snapshot. Numeric property
keys can change; the Terraform resource ID does not.

The entire edit set is validated before writing. Updates require vApp to be enabled,
unique IDs for changed properties, and a complete property snapshot. A changed key
also present in `deploy_options` is refused because creation gives that attribute
precedence. A previously managed property that is missing, an unmanaged existing
property, or an unexpected live value requires reconciliation before updating.
Unchanged legacy keys acquire no new ownership, and no validation beyond one
thing: before the provider starts a machine, any key present in the declared or
the previously declared map is checked for duplicate vApp entries, whoever
created them (see below).

The API trims and HTML-escapes strings. Empty values, leading/trailing whitespace,
and characters such as `/`, `&`, `<`, `>`, quotes, backticks and backslashes cannot
be sent faithfully through this endpoint and are rejected before any mutation.
This applies to IDs and preserved metadata too. Use the image's supported encoding
where appropriate, but note that ordinary Base64 can itself contain `/`; it is
not universally safe. Unsupported metadata or non-string property types cannot be
replaced without loss and are refused. Unrelated properties remain untouched.
These restrictions apply only when requesting a cloud-init update, not on creation
or when retaining an existing map.

## Failure and recovery

This is a non-atomic operation and requires an exclusive writer. Do not run it
while another VM operation, including an unresolved earlier request, is pending.
On an ordinary error the provider retains the prior `cloud_init` map and VM ID in
Terraform state. This preserves outstanding intent, **not a backend rollback**.
Some properties may have changed, or duplicate IDs may remain. A process killed
before returning state cannot be covered by this error handler.

When the provider powered the machine off itself, what happens next depends on
how far the update got. The four cases are distinct and are treated differently:

- **The shutdown was never confirmed** — the power-off call failed, the machine
  could not be re-read afterwards, or it still did not report itself stopped.
  Nothing was written, and the provider does **not** try to start the machine
  again: the platform can accept a power request whose response is lost, so a
  power-on here could race a shutdown that is still settling. Check the
  machine's power state yourself before re-running apply.
- **The shutdown was confirmed and nothing was written** — for example the vApp
  changed while the machine was going down. Nothing this apply did touched it
  — but that alone does not make it safe to start, so the provider gives it back
  only once the safety rule below is satisfied, confirms the result with a live
  read, and reports what it actually observed rather than what the activity
  claimed. The original
  error is still returned and the apply still fails.
- **A vApp write was attempted**, whatever outcome it reported. The machine is
  left **stopped** and the procedure below applies. The provider will not power
  it on, because a lost response still mutates the vApp and the properties may
  hold a duplicate or a half-applied edit. This is the one point where this path
  deliberately differs from the sizing power-cycle that shares the same
  `allow_vm_restart` flag: a sizing change is a single atomic PATCH and can be
  compensated by powering back on, and this one cannot.
- **The transaction was verified and a LATER step failed** — the general read
  of the machine, or the power settlement. The vApp holds the declared values,
  so Terraform **keeps** them in state. Rolling that intent back would be
  permanent, not merely untidy: `cloud_init` is never hydrated from the machine,
  so a later apply would see an added key as one it must refuse to adopt, or a
  removed key as one that has gone missing, with nothing left to repair it.
  Only the remaining state is left to converge — re-run apply, or set the power
  state by hand. These failures say "applied and verified" and deliberately do
  **not** print the recovery procedure above: the properties are already correct
  and, unless another writer has since changed the vApp, there is nothing to
  reconcile. If one has, the machine is left stopped and the diagnostic says so
  — the applied values are still kept.

Settling the power state at the end follows the same rule as the shutdown: the
machine must be observably running or stopped. A transitional or unrecognised
state is not evidence of the opposite stable state, so the provider leaves the
power state untouched and says so rather than issuing a power action that could
race an operation still settling.

More generally, on this update path the provider never **starts** a machine
without positive
evidence that starting it is safe. "This apply wrote nothing" is not that
evidence: another writer can have left a duplicate property behind in the
meantime. Before any power-on — settling the declared state, or giving back a
machine it stopped itself — it requires all three of: the machine observably
stopped, no duplicate entry for any `cloud_init` key this apply manages, and a
vApp still identical to the one it verified. If any of those fails the machine
is deliberately left stopped, with a diagnostic saying which, and reconciling
the vApp is yours to do. Powering a machine **off** needs none of this:
stopping a guest cannot boot it into a half-written vApp.

After a mutation, wait or verification failure:

1. Keep the VM stopped. Do not reapply or power it on.
2. Establish that all relevant backend activities and VMware tasks have finished,
   including requests whose response was lost. Use platform support if the outcome
   cannot be established.
3. Inspect and reconcile the actual descriptors. Preserve metadata and unrelated
   properties; resolve duplicates by numeric key. The original descriptor is retained
   until its replacement has been verified. A completed removal followed by a failed
   read can leave a missing property that the next update deliberately refuses.
4. Reconcile configuration/state with the settled result, then review a fresh plan.

There is no automatic mutation retry, rollback, durable operation journal or
cross-process lock. A matching snapshot cannot prove an earlier asynchronous
request has finished. Do not retry merely because the next plan still has a diff.
For fully automatic recovery the API would need durable idempotent operations and
outcome lookup; an atomic edit by stable key would additionally remove the duplicate
window.

## Drift detection

A refresh resynchronises `cloud_init` from the live vApp, so a value changed
outside Terraform shows up as a plan instead of staying invisible.

It covers **only the keys already recorded in state**, and that boundary is
deliberate. A machine's vApp carries far more than what Terraform declared —
properties from the image it was deployed from, from `deploy_options`, and from
the platform. Adopting every property whose id happens to be a cloud-init key
would pull values the operator never declared into state, and the next plan
would then propose to delete them. So:

- a **tracked** key whose live value changed is updated in state, and the next
  plan proposes to put it back;
- a **tracked** key that no longer exists on the machine is dropped from state,
  and the next plan proposes to add it;
- an **untracked** property is never adopted, whatever its id;
- a machine that declares no `cloud_init` at all is not affected, and its vApp
  is not even read.

A tracked key is removed from state only on strict positive evidence of absence
in a complete snapshot. An unreadable vApp, a duplicate entry for a tracked key,
or a non-string value all fail the refresh and leave the recorded values
untouched — dropping one would read as "Terraform never applied this".

This also repairs a state left behind by the old silent-update bug, where the
map recorded values that were never sent to the machine: the first refresh
records what the machine actually holds, and the next apply applies the
difference for real. An imported machine is not covered, because it records no
`cloud_init` for the refresh to resynchronise.
