package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// Host placement for cloudtemple_compute_iaas_opensource_virtual_machine (#355).
//
// Changing host_id to migrate a running intra-pool OpenIaaS VM used to be a
// silent no-op (host_id only flowed into the boot-on-host power request when
// power_state changed), so the apply reported success while the VM never moved
// and the next plan re-diffed forever. The fix wires the dedicated relocate
// endpoint and PROVES convergence; it never silently succeeds when the live
// host does not match the explicitly requested host.
//
// Intent is taken from the raw configuration, never from the Optional+Computed
// value reported by the live API (a refreshed/computed host_id is not user
// intent — same doctrine as osAdapterTxConfigured / openIaasVMDesiredProperties).

type openIaaSHostPlacement int

const (
	// hostPlacementNone: no genuine, user-requested host change to act on.
	hostPlacementNone openIaaSHostPlacement = iota
	// hostPlacementRelocate: a running VM stays running and must be live-migrated now.
	hostPlacementRelocate
	// hostPlacementOnPowerOn: the VM is being powered on; the boot-on-host power
	// request places it, and convergence is verified afterwards.
	hostPlacementOnPowerOn
	// hostPlacementErrorEndsPoweredOff: an explicit host change whose VM ends
	// powered off — relocation requires a running VM and a powered-off VM has no
	// authoritative resident host to converge on, so we fail closed.
	hostPlacementErrorEndsPoweredOff
)

// hostPlacementInputs is everything the placement logic needs, captured from
// the ResourceData BEFORE any mutation or final read.
type hostPlacementInputs struct {
	oldHost        string
	newHost        string
	oldPower       string // prior power_state ("on"/"off"); "" on create
	newPower       string // desired power_state ("on"/"off")
	requestedHost  string // explicitly requested host (== newHost)
	desiredPower   string // desired power_state (== newPower)
	hostConfigured bool   // host_id explicitly set in the raw config
	isNewResource  bool
}

// livePlacement is the live host + normalized power_state ("on"/"off") read
// back from the platform.
type livePlacement struct {
	host  string
	power string
}

// openIaaSHostPlacementFuncs are the side-effecting operations the placement
// orchestration depends on. They are injected so the orchestration is unit
// testable with fakes (pattern: vpcFloatingIPBindingFuncs).
type openIaaSHostPlacementFuncs struct {
	currentPlacement func(ctx context.Context, id string) (livePlacement, error)
	relocate         func(ctx context.Context, id, hostID string) (string, error)
	waitActivity     func(ctx context.Context, activityID string) error
	runPowerBlock    func() error
}

// hostIDConfiguredRaw reports whether host_id is explicitly set in the user
// configuration. A null/unknown config, a non-object value, or an absent
// attribute all mean "not configured" (and must never panic).
func hostIDConfiguredRaw(raw cty.Value) bool {
	if raw.IsNull() || !raw.IsKnown() {
		return false
	}
	t := raw.Type()
	if !t.IsObjectType() || !t.HasAttribute("host_id") {
		return false
	}
	v := raw.GetAttr("host_id")
	return !v.IsNull() && v.IsKnown()
}

// decideOpenIaaSHostPlacement is the pure placement decision. power values are
// the schema-normalized "on"/"off" (never the live API "Running"/"Halted").
func decideOpenIaaSHostPlacement(oldHost, newHost, oldPower, newPower string, hostConfigured bool) openIaaSHostPlacement {
	if !hostConfigured || newHost == "" || oldHost == newHost {
		return hostPlacementNone
	}
	if newPower != "on" {
		return hostPlacementErrorEndsPoweredOff
	}
	if oldPower == "on" {
		return hostPlacementRelocate
	}
	return hostPlacementOnPowerOn
}

// openIaaSHostPlacementInputs builds the placement inputs from the ResourceData.
// It must be called before any mutation/read (it reads the desired/changed
// values and the raw config).
func openIaaSHostPlacementInputs(d *schema.ResourceData) hostPlacementInputs {
	oldHost, newHost := d.GetChange("host_id")
	oldPower, newPower := d.GetChange("power_state")
	return hostPlacementInputs{
		oldHost:        oldHost.(string),
		newHost:        newHost.(string),
		oldPower:       oldPower.(string),
		newPower:       newPower.(string),
		requestedHost:  newHost.(string),
		desiredPower:   newPower.(string),
		hostConfigured: hostIDConfiguredRaw(d.GetRawConfig()),
		isNewResource:  d.IsNewResource(),
	}
}

// hostPlacementPreflightError fails fast, BEFORE any mutation, when an explicit
// host_id change would leave the VM powered off (it cannot be stably honored).
// Pure on the captured inputs.
func hostPlacementPreflightError(in hostPlacementInputs) diag.Diagnostics {
	if decideOpenIaaSHostPlacement(in.oldHost, in.newHost, in.oldPower, in.newPower, in.hostConfigured) == hostPlacementErrorEndsPoweredOff {
		return diag.Errorf("cannot place OpenIaaS/XCP-ng virtual machine on host %q while power_state ends \"off\": same-cluster (intra-pool) host placement requires the VM to be running, and a powered-off VM has no stable resident host; set power_state = \"on\"", in.requestedHost)
	}
	return nil
}

// preflightOpenIaaSHostPlacement captures the inputs from the ResourceData and
// runs the preflight. Used at the top of Create (Update captures the inputs
// once and reuses them for the orchestration).
func preflightOpenIaaSHostPlacement(d *schema.ResourceData) diag.Diagnostics {
	return hostPlacementPreflightError(openIaaSHostPlacementInputs(d))
}

// applyOpenIaaSHostPlacement runs, in order: the relocate (for a running VM
// whose host genuinely changed, real updates only), the injected power block,
// then the convergence assertion. Convergence is REQUIRED whenever host_id is
// explicitly configured and the VM ends powered on: the live VM must actually
// be powered on AND sitting on the requested host, otherwise we fail closed.
//
// All side effects go through f so the whole control flow is mutation-proven in
// unit tests with a stateful fake.
func applyOpenIaaSHostPlacement(ctx context.Context, id string, in hostPlacementInputs, f openIaaSHostPlacementFuncs) error {
	placement := decideOpenIaaSHostPlacement(in.oldHost, in.newHost, in.oldPower, in.newPower, in.hostConfigured)

	if placement == hostPlacementErrorEndsPoweredOff {
		// Normally already rejected by preflightOpenIaaSHostPlacement before any
		// mutation; kept as a defensive fail-closed.
		return fmt.Errorf("cannot place OpenIaaS/XCP-ng virtual machine %s on host %q while power_state ends \"off\": host placement requires the VM to be running", id, in.requestedHost)
	}

	if placement == hostPlacementRelocate && !in.isNewResource {
		cur, err := f.currentPlacement(ctx, id)
		if err != nil {
			return err
		}
		// Positive live evidence (not stale Terraform state) that the VM is
		// running: intra-pool relocation requires a running VM. Under a stale
		// state (e.g. -refresh=false) the planned power may say "on" while the
		// VM is actually off — never issue a placement operation in that case.
		if cur.power != "on" {
			return fmt.Errorf("cannot migrate OpenIaaS virtual machine %s to host %q: the VM is not running (live power_state %q); intra-pool host migration requires a running VM", id, in.requestedHost, cur.power)
		}
		// Idempotent: skip the relocate if the VM is already on the requested
		// host (e.g. a property-change reboot already placed it there).
		if cur.host != in.requestedHost {
			activityID, err := f.relocate(ctx, id, in.requestedHost)
			if err != nil {
				return fmt.Errorf("failed to migrate OpenIaaS virtual machine %s to host %q: %w", id, in.requestedHost, err)
			}
			if err := f.waitActivity(ctx, activityID); err != nil {
				return fmt.Errorf("failed to migrate OpenIaaS virtual machine %s to host %q: %w", id, in.requestedHost, err)
			}
		}
	}

	if err := f.runPowerBlock(); err != nil {
		return err
	}

	// Convergence: the apply must not report success while the live state does
	// not match the explicitly requested placement (the #355 contract).
	if in.hostConfigured && in.requestedHost != "" && in.desiredPower == "on" {
		cur, err := f.currentPlacement(ctx, id)
		if err != nil {
			return err
		}
		if cur.power != "on" {
			return fmt.Errorf("OpenIaaS/XCP-ng virtual machine %s did not reach power_state \"on\" after the requested host placement to %q (live power_state %q)", id, in.requestedHost, cur.power)
		}
		if cur.host != in.requestedHost {
			return fmt.Errorf("OpenIaaS/XCP-ng host migration of virtual machine %s did not converge: live host %q != requested host %q", id, cur.host, in.requestedHost)
		}
	}

	return nil
}

// newOpenIaaSHostPlacementFuncs wires the orchestration to the real client.
// currentPlacement reads the live VM and normalizes its power_state to
// "on"/"off" (the API reports "Running"/"Halted").
func newOpenIaaSHostPlacementFuncs(c *client.Client, runPowerBlock func() error) openIaaSHostPlacementFuncs {
	return openIaaSHostPlacementFuncs{
		currentPlacement: func(ctx context.Context, id string) (livePlacement, error) {
			vm, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, id)
			if err != nil {
				return livePlacement{}, fmt.Errorf("failed to read virtual machine %s for host-placement convergence: %w", id, err)
			}
			if vm == nil {
				return livePlacement{}, fmt.Errorf("failed to read virtual machine %s for host-placement convergence: not found", id)
			}
			power := "off"
			if vm.PowerState == "Running" {
				power = "on"
			}
			return livePlacement{host: vm.Host.ID, power: power}, nil
		},
		relocate: func(ctx context.Context, id, hostID string) (string, error) {
			return c.Compute().OpenIaaS().VirtualMachine().Relocate(ctx, id, &client.RelocateOpenIaasVirtualMachineRequest{HostId: hostID})
		},
		waitActivity: func(ctx context.Context, activityID string) error {
			_, err := c.Activity().WaitForCompletion(ctx, activityID, getWaiterOptions(ctx))
			return err
		},
		runPowerBlock: runPowerBlock,
	}
}
