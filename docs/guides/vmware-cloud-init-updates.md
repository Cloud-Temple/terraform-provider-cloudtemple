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

Use separate Terraform applies:

1. Set `power_state = "off"` and apply. The VM must already be stopped before
   updating cloud-init; changing power and cloud-init in the same apply is refused.
2. Change only `cloud_init` and apply. Do not combine this with any other VM
   configuration change. Only changed map keys are managed by this operation.
3. If required, update `os_network_adapter.ip_address` separately and apply while
   stopped. The guest network configuration and a VPC static IP reservation are
   separate settings; changing cloud-init alone does not change the reservation.
4. Set `power_state = "on"` and apply after the property update has succeeded.

The provider does not execute guest commands or implicitly restart the VM.
Changing vApp properties does not itself guarantee that cloud-init will run again.
For Ubuntu OVF images, the declared `instance-id` distinguishes cloud-init
instances; changing it can cause first-boot modules to run again, including modules
unrelated to hostname/networking. Choose the intended value explicitly and verify
the image's behavior. See [Ubuntu's VMware configuration documentation](https://documentation.ubuntu.com/vmware/vmware-how-to/configure-your-vm/).
The provider never generates a new instance ID for an update.

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
Unchanged legacy keys do not acquire new validation or ownership.

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

`cloud_init` remains a last-applied intent map, not general drift detection. An
unchanged map is not refreshed from vApp properties. A state produced by the old
silent-update bug is not automatically repaired by installing this provider;
reconcile the recorded intent with the real VM before requesting another update.
