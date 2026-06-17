package provider

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
)

// TestDecideOpenIaaSHostPlacement pins the pure placement decision (#355).
func TestDecideOpenIaaSHostPlacement(t *testing.T) {
	tests := []struct {
		name                                 string
		oldHost, newHost, oldPower, newPower string
		hostConfigured                       bool
		want                                 openIaaSHostPlacement
	}{
		{"not configured is never acted on", "A", "B", "on", "on", false, hostPlacementNone},
		{"empty new host is none", "A", "", "on", "on", true, hostPlacementNone},
		{"no change is none", "A", "A", "on", "on", true, hostPlacementNone},
		{"running to running relocates", "A", "B", "on", "on", true, hostPlacementRelocate},
		{"empty old running stays relocate", "", "B", "on", "on", true, hostPlacementRelocate},
		{"off to on places on power-on", "A", "B", "off", "on", true, hostPlacementOnPowerOn},
		{"empty old off to on places on power-on", "", "B", "off", "on", true, hostPlacementOnPowerOn},
		{"on to off ends powered off -> error", "A", "B", "on", "off", true, hostPlacementErrorEndsPoweredOff},
		{"off to off ends powered off -> error", "A", "B", "off", "off", true, hostPlacementErrorEndsPoweredOff},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := decideOpenIaaSHostPlacement(tt.oldHost, tt.newHost, tt.oldPower, tt.newPower, tt.hostConfigured)
			if got != tt.want {
				t.Fatalf("decideOpenIaaSHostPlacement(%q,%q,%q,%q,%v) = %d, want %d",
					tt.oldHost, tt.newHost, tt.oldPower, tt.newPower, tt.hostConfigured, got, tt.want)
			}
		})
	}
}

// TestHostIDConfiguredRaw pins that host_id intent comes ONLY from an explicit,
// known, non-null raw-config attribute — never panicking on null/unknown/absent.
func TestHostIDConfiguredRaw(t *testing.T) {
	objType := cty.Object(map[string]cty.Type{"host_id": cty.String})
	tests := []struct {
		name string
		raw  cty.Value
		want bool
	}{
		{"null config", cty.NullVal(objType), false},
		{"unknown config", cty.UnknownVal(objType), false},
		{"non-object config", cty.StringVal("nope"), false},
		{"object without host_id attribute", cty.EmptyObjectVal, false},
		{"host_id null", cty.ObjectVal(map[string]cty.Value{"host_id": cty.NullVal(cty.String)}), false},
		{"host_id unknown", cty.ObjectVal(map[string]cty.Value{"host_id": cty.UnknownVal(cty.String)}), false},
		{"host_id set", cty.ObjectVal(map[string]cty.Value{"host_id": cty.StringVal("host-b")}), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := hostIDConfiguredRaw(tt.raw); got != tt.want {
				t.Fatalf("hostIDConfiguredRaw(%#v) = %v, want %v", tt.raw, got, tt.want)
			}
		})
	}
}

// fakePlacer is a STATEFUL fake of openIaaSHostPlacementFuncs. The live host is
// only updated to the requested host once BOTH relocate AND waitActivity have
// run (unless relocateNoop simulates a platform that ignores the request), so a
// deleted relocate/waitActivity call cannot be masked. afterPower simulates the
// power block's effect on the live placement (boot-on-host, or a "successful"
// activity that nonetheless leaves the VM off).
type fakePlacer struct {
	liveHost, livePower string

	relocateErr, waitErr, powerErr, currentErr error
	relocateNoop                               bool
	afterPower                                 func(f *fakePlacer)

	relocateTarget                                     string
	relocateCalls, waitCalls, powerCalls, currentCalls int
	order                                              []string
}

func (f *fakePlacer) funcs() openIaaSHostPlacementFuncs {
	return openIaaSHostPlacementFuncs{
		currentPlacement: func(ctx context.Context, id string) (livePlacement, error) {
			f.currentCalls++
			f.order = append(f.order, "current")
			if f.currentErr != nil {
				return livePlacement{}, f.currentErr
			}
			return livePlacement{host: f.liveHost, power: f.livePower}, nil
		},
		relocate: func(ctx context.Context, id, hostID string) (string, error) {
			f.relocateCalls++
			f.relocateTarget = hostID
			f.order = append(f.order, "relocate")
			if f.relocateErr != nil {
				return "", f.relocateErr
			}
			return "activity-relocate", nil
		},
		waitActivity: func(ctx context.Context, activityID string) error {
			f.waitCalls++
			f.order = append(f.order, "wait")
			if f.waitErr != nil {
				return f.waitErr
			}
			if !f.relocateNoop && f.relocateCalls > 0 {
				f.liveHost = f.relocateTarget // migration confirmed: live host moves
			}
			return nil
		},
		runPowerBlock: func() error {
			f.powerCalls++
			f.order = append(f.order, "power")
			if f.powerErr != nil {
				return f.powerErr
			}
			if f.afterPower != nil {
				f.afterPower(f)
			}
			return nil
		},
	}
}

func inputs(hostConfigured bool, oldHost, newHost, oldPower, newPower string, isNew bool) hostPlacementInputs {
	return hostPlacementInputs{
		oldHost: oldHost, newHost: newHost, oldPower: oldPower, newPower: newPower,
		requestedHost: newHost, desiredPower: newPower,
		hostConfigured: hostConfigured, isNewResource: isNew,
	}
}

func indexOf(s []string, v string) int {
	for i, x := range s {
		if x == v {
			return i
		}
	}
	return -1
}

func TestApplyOpenIaaSHostPlacement(t *testing.T) {
	ctx := context.Background()

	t.Run("ends powered off is rejected before any side effect", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "on"}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "off", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "ends \"off\"") {
			t.Fatalf("want an ends-off error, got: %v", err)
		}
		if f.relocateCalls != 0 || f.powerCalls != 0 || f.currentCalls != 0 {
			t.Fatalf("no side effect must run on reject; relocate=%d power=%d current=%d", f.relocateCalls, f.powerCalls, f.currentCalls)
		}
	})

	t.Run("running host change relocates, waits, then converges", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "on"}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err != nil {
			t.Fatalf("expected convergence, got error: %v", err)
		}
		if f.relocateCalls != 1 || f.waitCalls != 1 {
			t.Fatalf("relocate must run exactly once and be awaited; relocate=%d wait=%d", f.relocateCalls, f.waitCalls)
		}
		if f.relocateTarget != "B" {
			t.Fatalf("relocate target = %q, want B", f.relocateTarget)
		}
		if f.powerCalls != 1 {
			t.Fatalf("power block must run once, got %d", f.powerCalls)
		}
		if ir, ip := indexOf(f.order, "relocate"), indexOf(f.order, "power"); ir < 0 || ip < 0 || ir > ip {
			t.Fatalf("relocate must happen before the power block; order=%v", f.order)
		}
	})

	t.Run("relocate that does not converge fails closed", func(t *testing.T) {
		// Platform ignores the request: live host stays A.
		f := &fakePlacer{liveHost: "A", livePower: "on", relocateNoop: true}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "did not converge") {
			t.Fatalf("want a non-convergence error, got: %v", err)
		}
		if f.relocateCalls != 1 {
			t.Fatalf("relocate must have been attempted once, got %d", f.relocateCalls)
		}
	})

	t.Run("stale state says running but VM is live-off: no relocate, fail closed", func(t *testing.T) {
		// Plan/state says power on (oldPower) but the live VM is actually off
		// (e.g. -refresh=false + an out-of-band shutdown). A placement op must
		// NOT be issued without positive live evidence the VM is running.
		f := &fakePlacer{liveHost: "A", livePower: "off"}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "the VM is not running") {
			t.Fatalf("want a not-running fail-closed error, got: %v", err)
		}
		if f.relocateCalls != 0 {
			t.Fatalf("no relocate must be issued without positive live evidence, got %d", f.relocateCalls)
		}
	})

	t.Run("idempotent skip when already on the requested host", func(t *testing.T) {
		f := &fakePlacer{liveHost: "B", livePower: "on"}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err != nil {
			t.Fatalf("already-converged must succeed, got: %v", err)
		}
		if f.relocateCalls != 0 {
			t.Fatalf("relocate must be skipped when already on the requested host, got %d", f.relocateCalls)
		}
		if f.powerCalls != 1 {
			t.Fatalf("power block must still run, got %d", f.powerCalls)
		}
	})

	t.Run("power-on path converges when the VM lands on the requested host", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "off", afterPower: func(f *fakePlacer) { f.liveHost = "B"; f.livePower = "on" }}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "off", "on", false), f.funcs())
		if err != nil {
			t.Fatalf("power-on placement must converge, got: %v", err)
		}
		if f.relocateCalls != 0 {
			t.Fatalf("the off->on path must NOT relocate (boot-on-host places it), got %d", f.relocateCalls)
		}
		if f.powerCalls != 1 {
			t.Fatalf("power block must run once, got %d", f.powerCalls)
		}
	})

	t.Run("power activity 'succeeds' but VM stays off -> error", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "off", afterPower: func(f *fakePlacer) { f.liveHost = "B"; f.livePower = "off" }}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "off", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "did not reach power_state \"on\"") {
			t.Fatalf("want a power-not-on error, got: %v", err)
		}
	})

	t.Run("power-on path lands on the wrong host -> non-convergence error", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "off", afterPower: func(f *fakePlacer) { f.liveHost = "C"; f.livePower = "on" }}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "off", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "did not converge") {
			t.Fatalf("want a non-convergence error, got: %v", err)
		}
	})

	t.Run("create path (isNewResource) never relocates but still verifies placement", func(t *testing.T) {
		// Even if the decision were Relocate, a new resource must not relocate;
		// placement is done by the boot-on-host power block and then verified.
		f := &fakePlacer{liveHost: "A", livePower: "off", afterPower: func(f *fakePlacer) { f.liveHost = "B"; f.livePower = "on" }}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", true), f.funcs())
		if err != nil {
			t.Fatalf("create placement must converge, got: %v", err)
		}
		if f.relocateCalls != 0 {
			t.Fatalf("create (isNewResource) must never relocate, got %d", f.relocateCalls)
		}
		if f.powerCalls != 1 {
			t.Fatalf("power block must run once on create, got %d", f.powerCalls)
		}
	})

	t.Run("unconfigured host_id performs no placement and no convergence", func(t *testing.T) {
		// host_id not configured: never act, never assert convergence (even if the live host differs).
		f := &fakePlacer{liveHost: "A", livePower: "on"}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(false, "A", "B", "on", "on", false), f.funcs())
		if err != nil {
			t.Fatalf("unconfigured host_id must not error, got: %v", err)
		}
		if f.relocateCalls != 0 {
			t.Fatalf("unconfigured host_id must not relocate, got %d", f.relocateCalls)
		}
		if f.powerCalls != 1 {
			t.Fatalf("power block must still run, got %d", f.powerCalls)
		}
		if f.currentCalls != 0 {
			t.Fatalf("no convergence read must happen when host_id is unconfigured, got %d", f.currentCalls)
		}
	})

	t.Run("relocate error is surfaced", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "on", relocateErr: errors.New("boom")}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "failed to migrate") {
			t.Fatalf("want a surfaced relocate error, got: %v", err)
		}
	})

	t.Run("waitActivity error is surfaced", func(t *testing.T) {
		f := &fakePlacer{liveHost: "A", livePower: "on", waitErr: errors.New("boom")}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "failed to migrate") {
			t.Fatalf("want a surfaced wait error, got: %v", err)
		}
	})

	t.Run("power block error is surfaced", func(t *testing.T) {
		f := &fakePlacer{liveHost: "B", livePower: "on", powerErr: errors.New("boom")}
		err := applyOpenIaaSHostPlacement(ctx, "vm-1", inputs(true, "A", "B", "on", "on", false), f.funcs())
		if err == nil || !strings.Contains(err.Error(), "boom") {
			t.Fatalf("want a surfaced power-block error, got: %v", err)
		}
	})
}
