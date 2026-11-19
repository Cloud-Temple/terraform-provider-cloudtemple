# Updating VMware cloud-init

`cloudtemple_compute_virtual_machine.cloud_init` retains its existing map-of-strings
schema. Creation still passes these values to the Marketplace deployment options.
An update instead changes the VM's vApp properties using the VMware property API;
it does not replace the VM. Existing configurations with an unchanged `cloud_init`
map do not enter the new update path. There is no schema version change and no
state migration. A refresh does now resynchronise this attribute's recorded
values from the machine, within the bounds described under Drift detection.

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
refused at **plan** time, naming the attribute, so you learn before an apply
begins rather than once the update is under way; apply it separately. When the
other attribute's new value is not yet resolved at plan time — it comes from
another resource, say — the plan lets it through and the refusal arrives at
apply instead, since only then is there a value to judge. A change to a `ForceNew` attribute
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

## Declaring a key in both maps

`cloud_init` and `deploy_options` end up in the **same** deployment request, and
`deploy_options` takes precedence. A key declared in both therefore reaches the
machine with its `deploy_options` value, while state records the `cloud_init`
one as though it had been applied. Nothing used to report this.

Declaring a key in both is refused at **plan** time, on creation as well as on
update, and whether or not the key changes:

```
Error: user-data declared in both cloud_init and deploy_options: creation sends
the deploy_options value and the cloud_init one is never applied, so declare
each key in exactly one of the two maps
```

The refusal is scoped to the **overlap**, not to using both attributes:

- `cloud_init` alongside unrelated `deploy_options` is the ordinary case and
  stays valid;
- `deploy_options` may own a cloud-init key outright — `hostname`, say — as long
  as `cloud_init` does not also claim it. The update path simply does not manage
  that key;
- only a key present in **both** maps is refused.

`cloud_init` restricts its own keys to the seven cloud-init ones, so an overlap
is by construction a cloud-init key claimed by `deploy_options`.

### Resolving one

Remove the key from **`cloud_init`**, not from `deploy_options`. The two moves
are not equivalent:

- `deploy_options` is `ForceNew`, so editing it at all **replaces the machine**;
- `cloud_init` is not. Removing the key from it is an in-place change, and it
  deliberately does **not** delete the vApp property — `deploy_options` still
  declares that value and the machine keeps it.

So on an existing resource, drop the key from `cloud_init`: nothing about the
machine changes, because the value it holds was the `deploy_options` one all
along. Terraform merely stops recording the key twice.

If `cloud_init` is the map that should own the key, the key has to leave
`deploy_options`, and that is a replacement.

## Importing a machine that already carries cloud-init

A key the configuration declares is a key the configuration owns, even when the
machine already carries it. The declared value wins and **overwrites** the live
one.

This matters most after `terraform import`, which records no `cloud_init` at
all. Every declared key is therefore untracked, and on a machine deployed with
cloud-init every one of them already exists — so refusing to adopt them, as
earlier releases did, left no configuration that could converge. Not even one
declaring exactly the values the machine already held.

The overwrite is a replacement of the existing property, not a fresh one: its
descriptor is cloned and only the value is swapped, so metadata the image's OVF
provided survives. A declared value identical to the live one plans no write at
all.

> **The plan under-reports this.** `cloud_init` holds only keys already in
> state, and a refresh cannot hydrate a key the configuration declares but state
> does not track — the configuration is not available to it. So after an import
> the plan reads `+ cloud_init = { ... }`, a creation, while the apply replaces
> a value that was already on the machine and that the plan never showed you.
> **Read the machine's vApp properties before the first apply on an imported
> resource.**

Adoption also carries a lifecycle: an adopted key is managed from then on, so
removing it from the configuration later **deletes** the vApp property, exactly
as for a key Terraform created.

## API constraints

The public API provides add and delete operations, not an atomic edit. A
replacement first adds a property with the same textual ID and preserved metadata,
verifies that exactly that property was added, then deletes the original by its
numeric key. Duplicate IDs exist temporarily while the VM is stopped. All other
properties must remain unchanged in every verified snapshot. Numeric property
keys can change; the Terraform resource ID does not.

The entire edit set is validated before writing. Updates require vApp to be enabled,
unique IDs for changed properties, and a complete property snapshot. A key present in
`deploy_options` is never written or deleted by this path, because creation gives
that attribute precedence; declaring it in both maps is refused at plan time. A previously managed property that is missing, an unmanaged existing
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
  permanent for a REMOVED key, not merely untidy: the refresh never removes a
  key it can no longer observe, so a later apply would see it as a managed
  property that has gone missing and refuse, with nothing left to repair it. An
  added key would now recover on its own, since a declared key the machine
  already carries is adopted — but the rule covers both, because rolling back an
  intent the machine has accepted also hides a write that really happened.
  Only the remaining state is left to converge — re-run apply, or set the power
  state by hand. These failures say "applied and verified" and deliberately do
  **not** print the recovery procedure above: the properties are already correct
  and, unless another writer has since changed the vApp, there is nothing to
  reconcile. If one has, the machine is left stopped and the diagnostic says so
  — the applied values are still kept.

### What a failed apply records about power

Terraform persists the values a failed apply had *planned*, not what happened.
Left alone, an apply that deliberately leaves a machine stopped would record it
as running — the one state the provider knows it is not in.

So a failed `cloud_init` update records the **last power state a live read
confirmed**, and nothing when no read confirmed one. A transitional or
unrecognised state confirms nothing and records nothing: the last stable
observation stands. A refusal whose observation contradicts the planned value
does not freeze the state, because freezing it would keep exactly the answer
that read has ruled out.

In practice: when a failure leaves the machine stopped, the state says `off`,
and the next plan proposes to start it rather than reporting nothing to do.

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

It only ever **replaces a value**, on a key Terraform already tracks and that
this attribute actually owns. It never adds a key, never removes one, and never
fails. Each bound exists because crossing it breaks a configuration that has not
changed:

- an **untracked** property is never adopted, whatever its id. A machine's vApp
  also carries values from the image it was deployed from, from `deploy_options`
  and from the platform; pulling those into state would make the next plan
  propose to delete values the operator never declared.
- a **tracked** key missing from the vApp is **kept**, not dropped. Absence is
  not evidence of deletion: a machine created by clone or from scratch receives
  no deploy options at all, so it legitimately records keys its vApp never held.
  Dropping them would make an unchanged configuration plan — and on a running
  machine without `allow_vm_restart`, that plan is then refused outright.
- a key also declared in **`deploy_options`** is never resynchronised. Creation
  sends the `deploy_options` value for a key present in both maps, so the live
  property belongs to that attribute. Recording it here would leave a difference
  the update path refuses for collision, with no way to converge.
- an anomaly never **fails** the refresh. An unreadable vApp, duplicate entries
  for a tracked key, or a non-string value are reported as **warnings** and
  leave the recorded value in place. A failed refresh takes the whole plan with
  it and leaves no provider-side recovery, which is too blunt for a step that
  only corrects values. Degradation is per key: one unreadable key does not
  freeze the resynchronisation of the others.
- a machine that declares no `cloud_init` at all is not affected, and its vApp
  is not even read.

### What drift detection does not repair

It repairs a state left by the original silent-update defect **only where the
key exists on the machine**: creation did send it, a later plan recorded a new
value without sending it, and the refresh now restores what the machine holds so
the next apply applies the difference for real.

A key the machine **never received** is not repaired. A `cloud_init` declared on
a clone or a from-scratch creation is never sent — neither path carries deploy
options — yet it is recorded in state. The refresh keeps it, so the
configuration stays convergent and silent. Changing the value is what brings
such a key into the update path and onto the machine.

An imported machine records no `cloud_init`, so a refresh has nothing to
resynchronise and, deliberately, adopts nothing: adoption is a decision for an
apply, which has the configuration to decide it with. See *Importing a machine
that already carries cloud-init* above.
