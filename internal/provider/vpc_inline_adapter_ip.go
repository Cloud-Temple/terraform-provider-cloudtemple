package provider

import (
	"context"
	"sort"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// This file holds the helpers shared by the two Compute VM resources
// (cloudtemple_compute_virtual_machine and
// cloudtemple_compute_iaas_opensource_virtual_machine) for the `ip_address`
// argument of their INLINE os_network_adapter block — the VPC static IP a VM can
// be created with in a single apply.
//
// It is deliberately separate from vpc_network_adapter_ip.go, which serves the
// STANDALONE adapter resources: there, ip_address lives at the resource root and
// ensureVPCForIPAddress can read it with raw.GetAttr("ip_address"). Inside a
// nested list block the value is per-index, which needs its own extraction — and
// the decision logic (validateIPAddressTargetsVPC, vpcStaticIPToPush) is reused
// from that file rather than forked.

// osAdapterIPConfigured returns, keyed by the block's INDEX in the user
// configuration, the `ip_address` values EXPLICITLY set on os_network_adapter
// blocks. `raw` is d.GetRawConfig().
//
// The raw config is authoritative, and the merged d.Get() map is not:
//   - the inline block is Optional+Computed as a whole, so the post-create state
//     merge seeds sibling fields from the live adapter;
//   - helpers.UpdateMapItems overlays config onto the live flatten through
//     d.GetOk, which treats "" as unset — so an explicitly empty value and an
//     absent one are indistinguishable there.
//
// An unknown raw value cannot occur during apply and carries no address to push,
// so it is reported as unconfigured (fail-safe: no preflight, no write).
func osAdapterIPConfigured(raw cty.Value) map[int]string {
	configured := map[int]string{}
	if raw.IsNull() || !raw.IsKnown() {
		return configured
	}
	rawAdapters := raw.GetAttr("os_network_adapter")
	if rawAdapters.IsNull() || !rawAdapters.IsKnown() {
		return configured
	}
	for i, adapter := range rawAdapters.AsValueSlice() {
		if adapter.IsNull() || !adapter.IsKnown() {
			continue
		}
		v := adapter.GetAttr("ip_address")
		if v.IsNull() || !v.IsKnown() {
			continue
		}
		if ip := v.AsString(); ip != "" {
			configured[i] = ip
		}
	}
	return configured
}

// validateInlineAdapterIPsTargetVPC fails closed BEFORE any side effect when an
// inline os_network_adapter block sets `ip_address` while its `network_id` does
// not reference a VPC-backed network.
//
// This mirrors ensureVPCForIPAddress for the nested case, and for the same
// measured reason: the platform SILENTLY IGNORES a static IP on a non-VPC network
// — no registration, no error (verified live on the VM Instances surface, and the
// Compute standalone adapters already enforce the same precondition). Without
// this check the user's chosen address would be dead configuration they believe
// took effect.
//
// networkIDAt returns the declared network_id of the block at a given index;
// status reads a network's VPC-ness (injected, since it differs per surface).
func validateInlineAdapterIPsTargetVPC(
	ctx context.Context,
	configuredIPs map[int]string,
	networkIDAt func(index int) string,
	status networkVPCStatusFunc,
) diag.Diagnostics {
	for _, index := range sortedIndexes(configuredIPs) {
		ip := configuredIPs[index]
		networkID := networkIDAt(index)
		if networkID == "" {
			// No network to validate against: the platform resolves the adapter's
			// network from the template/item. An address cannot be targeted at an
			// unknown network, so refuse rather than push it blindly.
			return diag.Errorf("os_network_adapter[%d] sets ip_address %q but declares no network_id: a static IP can only be assigned on an explicitly declared VPC network", index, ip)
		}
		vpcBacked, found, err := status(ctx, networkID)
		if err != nil {
			return diag.Errorf("failed to read network %s to validate the ip_address of os_network_adapter[%d]: %s", networkID, index, err)
		}
		if diags := validateIPAddressTargetsVPC(ip, networkID, vpcBacked, found); diags != nil {
			return diags
		}
	}
	return nil
}

// rejectInlineAdapterIPSharedNetwork fails when a block sets `ip_address` while
// another declared block targets the SAME network.
//
// MEASURED on the live API (OpenIaaS VM-create, DEV 2026-08-20): the platform
// validates the requested address against the number of adapters attached to that
// network and refuses with
//
//	"An explicit ipAddress (10.0.0.238) was requested for network <id>, but 2
//	 network adapters are attached to it — one IP cannot be assigned to several
//	 adapters."
//
// The address is therefore scoped to a (VM, network) pair, not to an individual
// adapter. The platform's refusal is correct but arrives only after the create
// call; catching it here fails closed before any side effect, and says which
// blocks collide.
func rejectInlineAdapterIPSharedNetwork(configuredIPs map[int]string, networkIDAt func(index int) string, blockCount int) diag.Diagnostics {
	for _, index := range sortedIndexes(configuredIPs) {
		ip := configuredIPs[index]
		target := networkIDAt(index)
		if target == "" {
			continue
		}
		for other := 0; other < blockCount; other++ {
			if other == index {
				continue
			}
			if networkIDAt(other) != target {
				continue
			}
			return diag.Errorf(
				"os_network_adapter[%d] sets ip_address %q on network %s, but os_network_adapter[%d] targets that same network: the platform assigns an explicit address per (virtual machine, network) pair and refuses one address for several adapters on the same network. Put the addressed adapter on its own network, or drop ip_address.",
				index, ip, target, other,
			)
		}
	}
	return nil
}

// rejectInlineAdapterIPWithoutLiveAdapter fails when a block that sets
// `ip_address` has no adapter to apply it to.
//
// helpers.UpdateNestedMapItems sizes the state it writes from the LIVE adapter
// list, so a configured block beyond that count is silently discarded — it never
// reaches the state and no API call is ever made for it. Silently discarding a
// network_id is long-standing behaviour and is left alone; silently discarding a
// deliberately chosen static IP is not acceptable, because the user has no way to
// notice. The check is therefore scoped to blocks that set ip_address.
func rejectInlineAdapterIPWithoutLiveAdapter(configuredIPs map[int]string, liveAdapterCount int, virtualMachineID string) diag.Diagnostics {
	for _, index := range sortedIndexes(configuredIPs) {
		ip := configuredIPs[index]
		if index < liveAdapterCount {
			continue
		}
		return diag.Errorf(
			"os_network_adapter[%d] sets ip_address %q but virtual machine %s has only %d network adapter(s): the block has no adapter to apply it to and would be silently dropped from the state. Remove the extra block, or manage the extra adapter with a dedicated network adapter resource.",
			index, ip, virtualMachineID, liveAdapterCount,
		)
	}
	return nil
}

// preserveInlineAdapterIP carries the `ip_address` of a PREVIOUS state entry into
// the freshly flattened one.
//
// ip_address is write-only: the platform does not echo the registered VPC static
// IP on the adapter object — it is addressable only by MAC on the /vpc/v1 plane —
// so a read that rebuilds the block from the live flatten alone would blank the
// attribute on every refresh and leave a permanent diff. Carrying the previous
// value across is what keeps it stable. The recorded semantic is therefore "last
// applied intent", not "live truth": this attribute has no drift detection, which
// is the accepted trade for not putting a /vpc/v1 read on the refresh path of
// every VM (a PAT without the vpc_read scope would then fail to refresh at all).
//
// prevEntry is the state entry the read loop is iterating; flat is the map built
// from the live adapter (the Flatten* helpers return interface{}, so both are
// taken loosely and a shape that is not a map is passed through untouched rather
// than panicking on a read path).
func preserveInlineAdapterIP(prevEntry interface{}, flat interface{}) interface{} {
	flatMap, ok := flat.(map[string]interface{})
	if !ok {
		return flat
	}
	prev, ok := prevEntry.(map[string]interface{})
	if !ok {
		return flat
	}
	if ip, ok := prev["ip_address"].(string); ok && ip != "" {
		flatMap["ip_address"] = ip
	}
	return flatMap
}

// inlineAdapterIPDescription is the shared user-facing wording of the nested
// ip_address argument. Kept in one place so the two surfaces cannot drift apart
// in what they promise.
const inlineAdapterIPDescription = "The VPC static IP to assign to this adapter at creation. " +
	"Requires `network_id` to reference a VPC-backed network: the platform silently ignores the value on a plain network, " +
	"so setting it there is rejected before anything is created. It is also rejected when another `os_network_adapter` block " +
	"targets the same network, because the platform assigns an explicit address per (virtual machine, network) pair and cannot " +
	"give one address to several adapters. When omitted on a VPC network, the platform auto-assigns an address. " +
	"Write-only: it is never read back from the platform (the registration is addressable only by MAC on the VPC plane), " +
	"so the value recorded in the state is the last one applied, and an out-of-band change is not detected as drift."

// sortedIndexes returns the keys of an index-keyed map in ascending order.
//
// Go randomises map iteration, so a check that returns on its FIRST offending
// entry would name a different block from one run to the next. A diagnostic the
// user cannot reproduce is a bad diagnostic, and a test asserting on one is
// flaky — hence the deterministic order: the lowest offending index is always the
// one reported.
func sortedIndexes(m map[int]string) []int {
	out := make([]int, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Ints(out)
	return out
}

// validateInlineAdapterIPPreconditions runs, in one call, every ip_address
// precondition that must hold before ANY platform mutation on an UPDATE: the
// target must be VPC-backed, and no other declared block may target that same
// network.
//
// It reads the explicit intent from the raw config (only blocks that actually set
// ip_address) and the concrete target from the PLANNED block list, where
// network_id may legitimately come from the template rather than the
// configuration. It is a no-op while the resource is new, because Create already
// validated the same blocks before its POST and then tail-calls Update.
func validateInlineAdapterIPPreconditions(ctx context.Context, d *schema.ResourceData, status networkVPCStatusFunc) diag.Diagnostics {
	if d.IsNewResource() {
		return nil
	}
	configuredIPs := osAdapterIPConfigured(d.GetRawConfig())
	if len(configuredIPs) == 0 {
		return nil
	}
	planned, _ := d.Get("os_network_adapter").([]interface{})
	networkIDAt := func(index int) string {
		if index >= len(planned) {
			return ""
		}
		block, ok := planned[index].(map[string]interface{})
		if !ok {
			return ""
		}
		id, _ := block["network_id"].(string)
		return id
	}
	if diags := rejectInlineAdapterIPSharedNetwork(configuredIPs, networkIDAt, len(planned)); diags != nil {
		return refuseBeforeAnyMutation(d, diags)
	}
	return refuseBeforeAnyMutation(d, validateInlineAdapterIPsTargetVPC(ctx, configuredIPs, networkIDAt, status))
}

// inlineAdaptersNeedCollection decides whether the update must walk the inline
// os_network_adapter blocks at all.
//
// A `d.HasChange` gate alone is NOT sufficient, and the reason is subtle enough to
// be worth spelling out. ip_address is write-only, so the read path preserves the
// stored value; and a refusal late in an apply persists the PLANNED values
// (terraform-plugin-sdk/v2 partial-apply semantics). Combine the two and the state
// can end up claiming an address the platform never registered — at which point
// state == config, `HasChange` is false, the reconciliation is never entered, and
// the address is never applied. Not a transient inconsistency: a PERMANENT one.
//
// Collecting the blocks whenever ANY of them configures an ip_address closes that
// hole. The cost is the normal device-reconciliation reads (the VM, its disks and
// its adapters) plus one by-MAC read per addressed adapter — no PATCH, because the
// reconciliation compares the configured address against the LIVE one and emits
// nothing when they already agree.
func inlineAdaptersNeedCollection(adaptersChanged bool, configuredIPs map[int]string) bool {
	return adaptersChanged || len(configuredIPs) > 0
}

// withPriorInlineAdapterIPs returns the PLANNED os_network_adapter list with EVERY
// entry's `ip_address` replaced by its prior value, matched by adapter id. All other
// fields are left exactly as planned.
//
// Two deliberate choices, both driven by an asymmetry:
//
//   - only ip_address is reverted, never a whole entry. A failed VPC relocation may
//     follow a network or MAC patch of the same apply that DID succeed, so rolling an
//     entry back wholesale would erase a real change.
//   - EVERY addressed adapter is reverted, not just the one that failed. The
//     reconciliation loop aborts on the first failure, so any later addressed adapter
//     was never attempted and its planned address must not be recorded. Reverting one
//     whose address DID apply is harmless — the next plan shows a diff, the next apply
//     re-pushes, the platform already matches, the push is a no-op and the state
//     converges. Reverting too FEW is not harmless: an unattempted address left in the
//     state kills the diff, Update is never called again, and the address is never
//     applied. Over-reverting costs one no-op apply; under-reverting is permanent.
//
// An adapter absent from the prior state (newly created, relocation failed) has no
// applied intent to preserve, so its address is cleared — which keeps the next apply
// retryable.
func withPriorInlineAdapterIPs(planned, prior []interface{}) []interface{} {
	priorIPByID := map[string]string{}
	for _, entry := range prior {
		block, ok := entry.(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := block["id"].(string); id != "" {
			priorIPByID[id], _ = block["ip_address"].(string)
		}
	}
	out := make([]interface{}, 0, len(planned))
	for _, entry := range planned {
		block, ok := entry.(map[string]interface{})
		if !ok {
			out = append(out, entry)
			continue
		}
		copied := make(map[string]interface{}, len(block))
		for k, v := range block {
			copied[k] = v
		}
		id, _ := block["id"].(string)
		copied["ip_address"] = priorIPByID[id]
		out = append(out, copied)
	}
	return out
}

// restoreInlineAdapterIPsOnFailure puts the PRIOR ip_address of every inline adapter
// back before surfacing a relocation failure.
//
// Without it, a failed relocation is a PERMANENT divergence rather than a retryable
// one: the SDK persists the planned values, the read preserves them (ip_address is
// write-only), so state == config and Terraform sees no diff left — Update is never
// called again and the address is never applied. Unlike a pre-mutation refusal this
// cannot use d.Partial(true), because the network/MAC patch of the same apply may
// legitimately have succeeded and must stay recorded.
func restoreInlineAdapterIPsOnFailure(d *schema.ResourceData, diags diag.Diagnostics) diag.Diagnostics {
	if diags == nil {
		return nil
	}
	priorRaw, _ := d.GetChange("os_network_adapter")
	prior, _ := priorRaw.([]interface{})
	planned, _ := d.Get("os_network_adapter").([]interface{})
	if err := d.Set("os_network_adapter", withPriorInlineAdapterIPs(planned, prior)); err != nil {
		return append(diags, diag.Diagnostic{
			Severity: diag.Warning,
			Summary:  "the unapplied ip_address values could not be rolled back in the state",
			Detail:   "A VPC static IP was not applied, but restoring the previous values in the state failed: " + err.Error() + ". The state may claim an address the platform does not have; re-run the apply after correcting the error.",
		})
	}
	return diags
}

// refuseBeforeAnyMutation surfaces a validation refusal WITHOUT letting any of the
// PLANNED values reach the state.
//
// terraform-plugin-sdk/v2 seeds an Update's ResourceData with the planned values
// and persists whatever it holds when Update returns an error — its partial-apply
// semantics. So a validation that refuses before touching the platform would still
// record the rejected configuration. Observed live: a refused apply left
// ip_address = "10.255.255.254" in the state and the next plan proposed "-> null",
// asking to remove a value that was never applied. Write-only attributes make this
// worse than cosmetic: unlike `name`, `ip_address` is not corrected by a refresh
// (the platform never echoes it), so the false intent persists.
//
// d.Partial(true) is the exact remedy: in State(), every attribute is then read
// from the PRIOR state instead of the set values
// (helper/schema/resource_data.go — `if d.partial { source = getSourceState }`).
// The whole prior state is preserved, not just the attribute the check looked at,
// which is precisely what "refused before any side effect" means.
//
// It is therefore ONLY correct for a refusal that precedes every mutation, and it
// is deliberately used at the TOP-LEVEL PREFLIGHTS ONLY. It must never be called
// after a platform call has succeeded: that would hide a real change. Later refusal
// paths in this resource (for example the sizing change refused because
// allow_vm_restart is false) may or may not follow an earlier mutation depending on
// what else the plan touched, so wrapping them blindly would be unsound — they are
// deliberately left alone.
//
// The residual, stated rather than hidden: such a later refusal still leaves the
// planned values in the state, including a write-only ip_address. That is
// pre-existing behaviour for every attribute of this resource, and for ip_address it
// is self-healing rather than permanent — but only because TWO independent
// properties hold, and both are pinned by tests:
//
//  1. the reconciliation DECISION ignores the state entirely: the desired address
//     comes from the raw config and the current one from the live platform
//     (TestInlineAdapterIPReconciliationIsStateIndependent);
//  2. the reconciliation is REACHABLE even with no adapter diff, because
//     inlineAdaptersNeedCollection opens the gate whenever an ip_address is
//     configured (TestInlineAdaptersNeedCollection). Without (2), an optimistic
//     state would make state == config, suppress the diff, and the address would
//     never be applied — permanently.
//
// Property (2) holds whenever an update REACHES the device reconciliation — earlier
// preflights and errors can return before it. Terraform still has to call Update, so a
// failure whose only diff was the address itself must not leave that address in the
// state — otherwise nothing would ever bring Update back. That is what
// restoreInlineAdapterIPsOnFailure guarantees on the relocation-failure path.
func refuseBeforeAnyMutation(d *schema.ResourceData, diags diag.Diagnostics) diag.Diagnostics {
	if diags == nil {
		return nil
	}
	d.Partial(true)
	return diags
}
