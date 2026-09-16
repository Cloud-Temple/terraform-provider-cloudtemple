package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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

// inlineIPConflictFunc reports the static IP registration that ALREADY holds `ip`
// on the VPC private network behind `networkID`, or nil when the address is free.
// It is injected because each surface reaches the network through its own client.
//
// An error must mean "could not establish the truth", never "free": see
// rejectInlineAdapterIPAlreadyRegistered for why that distinction is load-bearing.
type inlineIPConflictFunc func(ctx context.Context, networkID, ip string) (holder *client.StaticIP, err error)

// rejectInlineAdapterIPAlreadyRegistered fails when a configured `ip_address` is
// already registered to somebody else on the target VPC private network.
//
// MEASURED on the live API (OpenIaaS marketplace deploy, DEV 2026-08-21, 25
// concurrent creates): a VM asked for 10.0.5.103, an address already registered to
// another tenant resource. The platform did NOT refuse. It created the VM, answered
// success, and simply DID NOT register the address — leaving the adapter with no
// VPC static IP at all while Terraform recorded 10.0.5.103 in the state.
//
// That is the worst shape a defect can take here, and the reason this check exists:
//   - nothing fails, so no diagnostic ever reaches the user;
//   - `ip_address` is write-only, so no refresh compares it to the platform;
//   - state == config, so no future plan re-enters the reconciliation.
//
// The divergence is therefore PERMANENT and INVISIBLE. Only a pre-flight can catch
// it, because after the create there is no signal left to detect.
//
// FAIL CLOSED on a read error, deliberately, matching the sibling VPC precondition.
// Treating an unreadable listing as "address free" would reintroduce exactly the
// silent divergence above, and the failure is transient and retryable — which a
// permanently wrong state is not.
//
// ownerVMID excuses the resource's OWN registration: on an update the address is
// legitimately already registered to this very VM, and refusing that would make
// every no-op apply fail. It is empty on a create, where nothing can be ours yet.
func rejectInlineAdapterIPAlreadyRegistered(
	ctx context.Context,
	configuredIPs map[int]string,
	networkIDAt func(index int) string,
	conflictOf inlineIPConflictFunc,
	ownerVMID string,
) diag.Diagnostics {
	if conflictOf == nil {
		return nil
	}
	for _, index := range sortedIndexes(configuredIPs) {
		ip := configuredIPs[index]
		networkID := networkIDAt(index)
		if networkID == "" {
			// validateInlineAdapterIPsTargetVPC already refuses this case with a
			// better diagnostic; nothing to add here.
			continue
		}
		holder, err := conflictOf(ctx, networkID, ip)
		if err != nil {
			return diag.Errorf("failed to verify whether ip_address %q is already registered on the VPC private network of network %s (os_network_adapter[%d]): %s. Refusing rather than risk requesting an address the platform would silently decline to register.", ip, networkID, index, err)
		}
		if holder == nil {
			continue
		}
		if ownerVMID != "" && holder.VirtualMachine != nil && holder.VirtualMachine.ID == ownerVMID {
			// Already ours: this is the steady state of an update, not a conflict.
			continue
		}
		return diag.Errorf("os_network_adapter[%d] requests ip_address %q, which is ALREADY registered on that VPC private network%s. The platform does not refuse this: it would create the resource, report success, and silently leave the adapter with no static IP while Terraform recorded %q — a divergence no later refresh can detect, because ip_address is write-only. Choose a free address.", index, ip, describeStaticIPHolder(holder), ip)
	}
	return nil
}

// describeStaticIPHolder renders who holds an address, without dumping the whole
// record: enough for the user to find it, nothing more.
func describeStaticIPHolder(holder *client.StaticIP) string {
	if holder == nil {
		return ""
	}
	out := ""
	if holder.Source != "" {
		out += fmt.Sprintf(", registered by %s", holder.Source)
	}
	if holder.VirtualMachine != nil && holder.VirtualMachine.ID != "" {
		out += fmt.Sprintf(" for virtual machine %s", holder.VirtualMachine.ID)
	}
	if holder.MacAddress != "" {
		out += fmt.Sprintf(" on MAC %s", holder.MacAddress)
	}
	if holder.FloatingIP != nil {
		out += " (a floating IP is bound to it)"
	}
	return out
}

// staticIPConflictChecker builds an inlineIPConflictFunc from a resolver of the VPC
// private network behind a Compute network and a STRICT listing of that network's
// static IPs.
//
// The listing must be strict (client ListStrict): a partial or unprovable answer
// has to surface as an error. A lenient listing that returns an empty slice for a
// truncated body would read as "address free" and defeat the whole check.
//
// Listings are memoised per private network for the duration of one validation
// pass: several blocks may resolve to the same private network, and re-reading it
// would multiply calls against an API this very test proved fragile under load.
func staticIPConflictChecker(
	privateNetworkOf func(ctx context.Context, networkID string) (string, error),
	listStrict func(ctx context.Context, privateNetworkID string) ([]*client.StaticIP, error),
) inlineIPConflictFunc {
	cache := map[string][]*client.StaticIP{}
	return func(ctx context.Context, networkID, ip string) (*client.StaticIP, error) {
		pnID, err := privateNetworkOf(ctx, networkID)
		if err != nil {
			return nil, err
		}
		if pnID == "" {
			// Not VPC-backed, or the platform does not expose the link. The VPC
			// precondition owns that verdict; there is no IPAM plane to check.
			return nil, nil
		}
		rows, cached := cache[pnID]
		if !cached {
			rows, err = listStrict(ctx, pnID)
			if err != nil {
				return nil, err
			}
			cache[pnID] = rows
		}
		for _, row := range rows {
			if row != nil && row.IPAddress == ip {
				return row, nil
			}
		}
		return nil, nil
	}
}

// inlineAdapterIPCollisionDiff is the PLAN-TIME half of the IPAM collision check.
//
// The create/update preconditions already refuse a taken address before any
// platform call, which is what protects the state. But they run during APPLY, so a
// `terraform plan` looks clean and the user only learns at apply time. Catching it
// in CustomizeDiff turns it into a plan error, which is where a mistake costs the
// least.
//
// It is DELIBERATELY tolerant of unknowns and never authoritative:
//   - a network_id still unknown at plan time (computed, or sourced from the
//     template rather than the config) is skipped — a plan must not fail because a
//     value has not been resolved yet;
//   - the apply-time check remains the real gate, because the IPAM plane can change
//     between plan and apply and only the pre-create check is ordered against the
//     platform call.
//
// So this can produce a false PASS, never a false FAIL.
func inlineAdapterIPCollisionDiff(conflictOf inlineIPConflictFunc) func(ctx context.Context, diff *schema.ResourceDiff, meta any) error {
	return func(ctx context.Context, diff *schema.ResourceDiff, meta any) error {
		if conflictOf == nil {
			// No client to read the IPAM plane with (unit tests drive Resource.Diff
			// with a nil meta). Skip rather than panic: this hook is advisory and the
			// create/update precondition is the real gate.
			return nil
		}
		configuredIPs := osAdapterIPConfigured(diff.GetRawConfig())
		if len(configuredIPs) == 0 {
			return nil
		}
		blocks, _ := diff.Get("os_network_adapter").([]interface{})
		networkIDAt := func(index int) string {
			if index >= len(blocks) {
				return ""
			}
			block, ok := blocks[index].(map[string]interface{})
			if !ok {
				return ""
			}
			id, _ := block["network_id"].(string)
			return id
		}
		diags := rejectInlineAdapterIPAlreadyRegistered(ctx, configuredIPs, networkIDAt, conflictOf, diff.Id())
		if diags.HasError() {
			return fmt.Errorf("%s", diags[0].Summary)
		}
		return nil
	}
}

// inlineIPConflictOrNil returns nil when meta carries no client, so the plan-time
// hook degrades to a no-op instead of panicking. Production always has one.
func inlineIPConflictOrNil(meta any, build func(*client.Client) inlineIPConflictFunc) inlineIPConflictFunc {
	c, ok := meta.(*client.Client)
	if !ok || c == nil {
		return nil
	}
	return build(c)
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
const inlineAdapterIPDescription = "The VPC static IP to assign to this adapter. It is applied when the virtual machine is created, " +
	"and changing it later relocates the address in place (no replacement). " +
	"Requires `network_id` to reference a VPC-backed network: the platform silently ignores the value on a plain network, " +
	"so setting it there is rejected before anything is created or changed. It is also rejected when another `os_network_adapter` block " +
	"targets the same network, because the platform assigns an explicit address per (virtual machine, network) pair and cannot " +
	"give one address to several adapters. It is also rejected when the address is ALREADY registered on the target VPC " +
	"private network: the platform does not refuse that case — it creates the resource, reports success and silently " +
	"registers nothing — so the collision is checked before anything is created. " +
	"When omitted on a VPC network, the platform auto-assigns an address. " +
	"Not supported on a deployment mode that provides no network adapter for it to apply to — a VMware from-scratch create " +
	"(`guest_operating_system_moref`) — which is rejected with an explicit error rather than silently ignored. " +
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
func validateInlineAdapterIPPreconditions(ctx context.Context, d *schema.ResourceData, status networkVPCStatusFunc, conflictOf inlineIPConflictFunc) diag.Diagnostics {
	if d.IsNewResource() {
		// Create validated the same blocks before its own calls and then tail-calls
		// Update; re-reading the networks here would buy nothing.
		return nil
	}
	// On an UPDATE the adapters already exist, so a configured address whose block
	// carries no adapter id has nothing to be applied to. Left unchecked the apply
	// SUCCEEDS while doing nothing and the read then drops the block, so the plan
	// never converges and the user is never told why.
	if diags := rejectInlineAdapterIPWithoutAdapterID(osAdapterIPConfigured(d.GetRawConfig()), plannedInlineAdapters(d)); diags != nil {
		return refuseBeforeAnyMutation(d, diags)
	}
	return validateInlineAdapterIPPreconditionsCore(ctx, d, status, conflictOf, d.Id())
}

// validateInlineAdapterIPPreconditionsOnCreate is the CREATE-side entry point.
// d.IsNewResource() is precisely the case to validate here, so there is no skip; and
// it additionally rejects an address on a deployment mode that produces NO adapter
// at all, before anything is created.
func validateInlineAdapterIPPreconditionsOnCreate(ctx context.Context, d *schema.ResourceData, status networkVPCStatusFunc, conflictOf inlineIPConflictFunc, adapterlessMode bool, modeName string) diag.Diagnostics {
	if diags := rejectInlineAdapterIPOnAdapterlessMode(osAdapterIPConfigured(d.GetRawConfig()), adapterlessMode, modeName); diags != nil {
		return refuseBeforeAnyMutation(d, diags)
	}
	// Empty owner: on a create nothing on the IPAM plane can be ours yet, so ANY
	// existing registration of the requested address is a conflict.
	return validateInlineAdapterIPPreconditionsCore(ctx, d, status, conflictOf, "")
}

// validateInlineAdapterIPPreconditionsCore holds the checks both entry points share.
//
// refuseBeforeAnyMutation is applied to the verdict, so a refusal never records the
// planned values: every call site is placed before any platform call.
func validateInlineAdapterIPPreconditionsCore(ctx context.Context, d *schema.ResourceData, status networkVPCStatusFunc, conflictOf inlineIPConflictFunc, ownerVMID string) diag.Diagnostics {
	configuredIPs := osAdapterIPConfigured(d.GetRawConfig())
	if len(configuredIPs) == 0 {
		return nil
	}
	planned := plannedInlineAdapters(d)
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
	if diags := validateInlineAdapterIPsTargetVPC(ctx, configuredIPs, networkIDAt, status); diags != nil {
		return refuseBeforeAnyMutation(d, diags)
	}
	// Ordered last on purpose: it costs a listing per targeted private network, so
	// it only runs once the cheaper structural checks have passed.
	return refuseBeforeAnyMutation(d, rejectInlineAdapterIPAlreadyRegistered(ctx, configuredIPs, networkIDAt, conflictOf, ownerVMID))
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

// rollbackInlineAdapterIPsOnAnyError is installed with `defer` at the top of a
// create or update function, against its NAMED return value:
//
//	func fooUpdate(...) (diags diag.Diagnostics) {
//	    defer rollbackInlineAdapterIPsOnAnyError(d, &diags)
//
// It exists to make a state-safety property STRUCTURAL instead of a discipline that
// has to be remembered at every error return.
//
// The hazard, once more, because it is the whole reason this exists: `ip_address` is
// write-only, so the read path preserves whatever the state holds; and
// terraform-plugin-sdk/v2 persists the PLANNED values when a create or update
// returns an error. A single error return that does not roll the addresses back is
// therefore enough to record an address that was never applied — after which state
// equals config, Terraform sees no diff, the function is never called again, and the
// address is never applied. Permanent, and invisible in the plan.
//
// Wrapping each return individually was tried and does not hold: adversarial review
// found five separate forgotten spots, each one narrower than the last. A deferred
// guard inverts the burden — every present AND FUTURE error return is covered, and
// there is nothing left to forget.
//
// Rolling back on ANY error, including one raised after the address was successfully
// applied, is deliberate and safe. The reconciliation compares the configuration
// against the LIVE platform, never against the state, so an over-reverted address
// costs exactly one no-op apply to converge. Under-reverting costs correctness,
// permanently. Given that asymmetry, the unconditional rule is the right one, and it
// needs no flag tracking how far the function got.
func rollbackInlineAdapterIPsOnAnyError(d *schema.ResourceData, diags *diag.Diagnostics) {
	if diags == nil || !diags.HasError() {
		return
	}
	*diags = restoreInlineAdapterIPsOnFailure(d, *diags)
}

// plannedInlineAdapters is the planned os_network_adapter list, tolerating a nil or
// wrongly-typed value.
func plannedInlineAdapters(d *schema.ResourceData) []interface{} {
	planned, _ := d.Get("os_network_adapter").([]interface{})
	return planned
}

// rejectInlineAdapterIPWithoutAdapterID fails when a block that sets `ip_address`
// carries no adapter id.
//
// On an UPDATE every managed adapter already has one, so an empty id means the block
// has no adapter behind it. Both surfaces then skip it silently — VMware's update
// branches are both keyed on a non-empty id, and the read rebuilds the list from the
// live adapters — so the apply SUCCEEDS having done nothing, the block disappears
// from the state, and the plan never converges. The user gets no diagnostic at all.
// Refusing is the only outcome that tells them what is wrong.
func rejectInlineAdapterIPWithoutAdapterID(configuredIPs map[int]string, planned []interface{}) diag.Diagnostics {
	for _, index := range sortedIndexes(configuredIPs) {
		if index >= len(planned) {
			// Out of range is rejectInlineAdapterIPWithoutLiveAdapter's business.
			continue
		}
		block, ok := planned[index].(map[string]interface{})
		if !ok {
			continue
		}
		if id, _ := block["id"].(string); id == "" {
			return diag.Errorf(
				"os_network_adapter[%d] sets ip_address %q but has no network adapter behind it (no id in the state): the address cannot be applied to anything, and the block would be silently dropped on the next read. Remove ip_address, or manage the adapter with a dedicated network adapter resource.",
				index, configuredIPs[index],
			)
		}
	}
	return nil
}

// rejectInlineAdapterIPOnAdapterlessMode fails when an address is configured while
// the chosen deployment mode produces NO network adapter at all.
//
// On VMware a from-scratch create carries no network field whatsoever, so the address
// could never take effect. Unlike the live-adapter-count check, this is knowable from
// the configuration ALONE, before anything is created — so failing here spares the
// operator a virtual machine created for nothing and left to clean up.
func rejectInlineAdapterIPOnAdapterlessMode(configuredIPs map[int]string, adapterlessMode bool, modeName string) diag.Diagnostics {
	if !adapterlessMode {
		return nil
	}
	for _, index := range sortedIndexes(configuredIPs) {
		return diag.Errorf(
			"os_network_adapter[%d] sets ip_address %q, but a virtual machine created %s has no network adapter for it to apply to: the address could never take effect. Remove ip_address, or deploy from a source that provides adapters (clone, content library or marketplace item).",
			index, configuredIPs[index], modeName,
		)
	}
	return nil
}
