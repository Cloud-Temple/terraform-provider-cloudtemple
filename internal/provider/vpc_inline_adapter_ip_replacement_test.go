package provider

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
	ctyjson "github.com/hashicorp/go-cty/cty/json"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

// These tests pin issue #533: with provider 1.12.0, planning the REPLACEMENT of a
// cloudtemple_public_cloud_vm_instance that holds a VPC static ip_address failed with
// "ip_address ... is ALREADY registered ... for virtual machine <its own id>".
//
// Terraform plans a replacement in two PlanResourceChange calls. The second one — the
// create half — is made against a NULL prior state (terraform:
// internal/terraform/node_resource_abstract_instance.go, plan(), `if action.IsReplace()`;
// OpenTofu is identical), so terraform-plugin-sdk/v2 hands CustomizeDiff a ResourceDiff
// whose Id() is "" and whose raw state is null. The plan-time collision hook could not
// excuse the resource's own registration and reported it as a foreign conflict.
//
// The harness below feeds Resource.SimpleDiff EXACTLY what
// helper/schema/grpc_provider.go PlanResourceChange feeds it, so the null-prior call is
// reproduced offline rather than approximated with a hand-built state.

// recordingConflict is an inlineIPConflictFunc that records every IPAM lookup. Every
// expected-pass case asserts on the calls it did (or did not) make, so a raw config
// that failed to reach the hook could not make a case pass vacuously.
type recordingConflict struct {
	calls  []string // "networkID|ip", in call order
	holder *client.StaticIP
	err    error
}

func (r *recordingConflict) fn() inlineIPConflictFunc {
	return func(_ context.Context, networkID, ip string) (*client.StaticIP, error) {
		r.calls = append(r.calls, networkID+"|"+ip)
		return r.holder, r.err
	}
}

// ctyFromJSON builds a fully-typed value of the resource's implied type from a JSON
// document. Every attribute the document omits is null — exactly how Terraform sends
// an unset attribute — so the value conforms to the schema without spelling out the
// twenty-odd attributes each case does not care about.
func ctyFromJSON(t *testing.T, ty cty.Type, doc map[string]interface{}) cty.Value {
	t.Helper()
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal test document: %v", err)
	}
	v, err := ctyjson.Unmarshal(b, ty)
	if err != nil {
		t.Fatalf("the test document does not conform to the resource schema: %v", err)
	}
	return v
}

// proposedNewFor approximates Terraform's objchange.ProposedNew at the top level: the
// configuration, with every Computed attribute the configuration leaves null taken
// from the prior state when there is one, and unknown otherwise (a fresh create).
func proposedNewFor(res *schema.Resource, prior, config cty.Value) cty.Value {
	vals := map[string]cty.Value{}
	for name, aty := range config.Type().AttributeTypes() {
		v := config.GetAttr(name)
		s, declared := res.Schema[name]
		computed := name == "id" || (declared && s.Computed)
		if computed && v.IsNull() {
			if !prior.IsNull() && !prior.GetAttr(name).IsNull() {
				v = prior.GetAttr(name)
			} else {
				v = cty.UnknownVal(aty)
			}
		}
		vals[name] = v
	}
	return cty.ObjectVal(vals)
}

// planShim is what PlanResourceChange builds before calling Resource.SimpleDiff:
// the prior state shimmed from its cty value (an InstanceState with an EMPTY id when
// the value is null — a create, or the create half of a replacement), carrying the raw
// state, plan and config, plus the legacy ResourceConfig built from the PROPOSED new
// state (not from the configuration).
func planShim(t *testing.T, res *schema.Resource, prior, config cty.Value) (*terraform.InstanceState, *terraform.ResourceConfig) {
	t.Helper()
	proposed := proposedNewFor(res, prior, config)
	priorState, err := res.ShimInstanceStateFromValue(prior)
	if err != nil {
		t.Fatalf("ShimInstanceStateFromValue: %v", err)
	}
	priorState.RawState = prior
	priorState.RawPlan = proposed
	priorState.RawConfig = config
	return priorState, terraform.NewResourceConfigShimmed(proposed, res.CoreConfigSchema())
}

// planResource runs the protocol entry point (SimpleDiff) on the shimmed inputs.
func planResource(t *testing.T, res *schema.Resource, prior, config cty.Value) (*terraform.InstanceDiff, error) {
	t.Helper()
	priorState, cfg := planShim(t, res, prior, config)
	return res.SimpleDiff(context.Background(), priorState, cfg, nil)
}

const (
	vmiSelfID       = "vm-self"
	vmiNetworkID    = "55555555-5555-5555-5555-555555555555"
	vmiStaticIP     = "10.0.6.240"
	vmiExpectedIPAM = vmiNetworkID + "|" + vmiStaticIP
)

// vmiDocument is the reporter's configuration: one inline adapter on a VPC network
// with an explicit ip_address, VM declared stopped. withCloudInit adds the ForceNew
// attribute whose change triggers the replacement.
func vmiDocument(withCloudInit bool) map[string]interface{} {
	doc := map[string]interface{}{
		"name":                 "control-01",
		"availability_zone_id": "11111111-1111-1111-1111-111111111111",
		"image_id":             "22222222-2222-2222-2222-222222222222",
		"instance_family_id":   "33333333-3333-3333-3333-333333333333",
		"cpu":                  2,
		"memory":               4,
		"backup_policy_id":     "44444444-4444-4444-4444-444444444444",
		"power_state":          "off",
		"os_network_adapter": []interface{}{
			map[string]interface{}{"device_index": 0, "network_id": vmiNetworkID, "ip_address": vmiStaticIP},
		},
	}
	if withCloudInit {
		doc["cloud_init"] = map[string]interface{}{"cloud_config": "#cloud-config\nhostname: control-01\n"}
	}
	return doc
}

// vmiPriorDocument is the recorded state of that VM: the configuration plus its id
// and the computed attributes a read fills in.
func vmiPriorDocument() map[string]interface{} {
	doc := vmiDocument(false)
	doc["id"] = vmiSelfID
	doc["status"] = "stopped"
	return doc
}

func staticIPHeldBy(vmID, source string) *client.StaticIP {
	return staticIPHeldByMAC(vmID, source, "3a:ad:76:5d:e4:e9")
}

// staticIPHeldByMAC builds a registration with an explicit MAC; an empty vmID leaves
// the machine link nil — the MAC-owned shape of the Compute surfaces.
func staticIPHeldByMAC(vmID, source, mac string) *client.StaticIP {
	s := &client.StaticIP{IPAddress: vmiStaticIP, Source: source, MacAddress: mac}
	if vmID != "" {
		s.VirtualMachine = &client.BaseObject{ID: vmID}
	}
	return s
}

func assertCalls(t *testing.T, rec *recordingConflict, want ...string) {
	t.Helper()
	if len(rec.calls) != len(want) {
		t.Fatalf("IPAM lookups = %v, want %v", rec.calls, want)
	}
	for i := range want {
		if rec.calls[i] != want[i] {
			t.Fatalf("IPAM lookup %d = %q, want %q (all: %v)", i, rec.calls[i], want[i], rec.calls)
		}
	}
}

// TestVMInstancePlanTimeIPCollision_Replacement is the #533 regression test, driven
// through the real cloudtemple_public_cloud_vm_instance resource.
func TestVMInstancePlanTimeIPCollision_Replacement(t *testing.T) {
	newRes := func(rec *recordingConflict) *schema.Resource {
		res := resourcePublicCloudVMInstance()
		res.CustomizeDiff = customizeVMInstanceDiffWith(rec.fn())
		return res
	}
	ty := resourcePublicCloudVMInstance().CoreConfigSchema().ImpliedType()

	t.Run("in-place plan: the VM's own registration is not a conflict", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy(vmiSelfID, "vmi")}
		res := newRes(rec)
		prior := ctyFromJSON(t, ty, vmiPriorDocument())
		config := ctyFromJSON(t, ty, vmiDocument(false))
		if _, err := planResource(t, res, prior, config); err != nil {
			t.Fatalf("a no-op plan on the VM that holds the address must pass, got: %v", err)
		}
		// The identity path DID consult IPAM — the pass is not a short-circuit.
		assertCalls(t, rec, vmiExpectedIPAM)
	})

	t.Run("in-place plan: an address held by ANOTHER VM is refused at plan (unchanged behaviour)", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy("someone-else", "vmi")}
		res := newRes(rec)
		prior := ctyFromJSON(t, ty, vmiPriorDocument())
		config := ctyFromJSON(t, ty, vmiDocument(false))
		_, err := planResource(t, res, prior, config)
		if err == nil {
			t.Fatal("an address registered to another VM must still fail the plan when the plan carries an identity")
		}
		for _, want := range []string{"ALREADY registered", "someone-else", vmiStaticIP} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("plan error is missing %q: %v", want, err)
			}
		}
		// The replacement hint is a CREATE-time hint: with an identity the refusal
		// is about somebody else's address, and must not suggest a replacement retry.
		if strings.Contains(err.Error(), "create_before_destroy") {
			t.Fatalf("the replacement-ordering hint must not appear on an in-place refusal: %v", err)
		}
		assertCalls(t, rec, vmiExpectedIPAM)
	})

	t.Run("in-place plan: an unreadable IPAM plane fails the plan closed", func(t *testing.T) {
		rec := &recordingConflict{err: errors.New("vpc plane unreachable")}
		res := newRes(rec)
		prior := ctyFromJSON(t, ty, vmiPriorDocument())
		config := ctyFromJSON(t, ty, vmiDocument(false))
		if _, err := planResource(t, res, prior, config); err == nil {
			t.Fatal("an IPAM read error must fail the plan: unreadable never means free")
		}
		assertCalls(t, rec, vmiExpectedIPAM)
	})

	t.Run("replacement, pass 1 (prior state present): adding cloud_init plans a replacement and passes", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy(vmiSelfID, "vmi")}
		res := newRes(rec)
		prior := ctyFromJSON(t, ty, vmiPriorDocument())
		config := ctyFromJSON(t, ty, vmiDocument(true))
		diff, err := planResource(t, res, prior, config)
		if err != nil {
			t.Fatalf("the first pass of the replacement plan must pass (own registration): %v", err)
		}
		if diff == nil || !diff.RequiresNew() {
			t.Fatalf("adding cloud_init must plan a REPLACEMENT (ForceNew); got diff %+v", diff)
		}
		assertCalls(t, rec, vmiExpectedIPAM)
	})

	// This is the exact call that failed in #533. Terraform re-plans the create half
	// of the replacement against a NULL prior state; the provider sees no id, and the
	// address is held by the very VM about to be destroyed. RED on v1.12.0.
	t.Run("replacement, pass 2 (NULL prior state): the VM's own registration must not fail the plan (#533)", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy(vmiSelfID, "vmi")}
		res := newRes(rec)
		config := ctyFromJSON(t, ty, vmiDocument(true))
		if _, err := planResource(t, res, cty.NullVal(ty), config); err != nil {
			t.Fatalf("#533: the create half of a replacement must not report the VM's own registration as a conflict, got: %v", err)
		}
		// Without an identity the hook cannot adjudicate, so it must not even ask:
		// the verdict belongs to the pre-create check.
		assertCalls(t, rec)
	})

	// A fresh create is indistinguishable from the case above (same request shape), so
	// it receives the same deferral: the pre-create check refuses it before anything
	// is created. This pins the deliberate trade — a false PASS at plan is allowed, a
	// false FAIL is not.
	t.Run("no identity: an address held by another VM is deferred to the pre-create check", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy("someone-else", "vmi")}
		res := newRes(rec)
		config := ctyFromJSON(t, ty, vmiDocument(false))
		if _, err := planResource(t, res, cty.NullVal(ty), config); err != nil {
			t.Fatalf("without an identity the plan-time hook must defer, got: %v", err)
		}
		assertCalls(t, rec)
	})

	// The SDK has its own replacement re-diff: Resource.Diff (handleRequiresNew=true,
	// the offline path this repository's plan tests use) re-runs CustomizeDiff with a
	// nil state after a RequiresNew. Same symptom, same fix.
	t.Run("offline Resource.Diff replacement re-diff: the state-less second pass does not fail", func(t *testing.T) {
		rec := &recordingConflict{holder: staticIPHeldBy(vmiSelfID, "vmi")}
		res := newRes(rec)
		prior := ctyFromJSON(t, ty, vmiPriorDocument())
		config := ctyFromJSON(t, ty, vmiDocument(true))
		priorState, cfg := planShim(t, res, prior, config)
		diff, err := res.Diff(context.Background(), priorState, cfg, nil)
		if err != nil {
			t.Fatalf("Resource.Diff must pass on a replacement of the VM that holds the address: %v", err)
		}
		if diff == nil || !diff.RequiresNew() {
			t.Fatalf("expected a replacement diff, got %+v", diff)
		}
		// One lookup from the pass that had an identity; none from the re-diff.
		assertCalls(t, rec, vmiExpectedIPAM)
	})
}

// TestComputeInlineAdapterPlanTimeIPCollision_NoIdentity runs the shared hook against
// the REAL os_network_adapter schemas of the two Compute VM resources, which wire the
// same hook and have ForceNew attributes of their own. Their production CustomizeDiff
// chains take the IPAM view from the client in meta, so the hook is mounted alone on a
// copy of each schema; what this guards is the hook's behaviour against each surface's
// actual block shape.
func TestComputeInlineAdapterPlanTimeIPCollision_NoIdentity(t *testing.T) {
	surfaces := []struct {
		name   string
		schema map[string]*schema.Schema
		source string
	}{
		{"cloudtemple_compute_virtual_machine", resourceVirtualMachine().Schema, "vmware"},
		{"cloudtemple_compute_iaas_opensource_virtual_machine", resourceOpenIaasVirtualMachine().Schema, "xoa"},
	}
	for _, sf := range surfaces {
		t.Run(sf.name, func(t *testing.T) {
			newRes := func(rec *recordingConflict) *schema.Resource {
				return &schema.Resource{Schema: sf.schema, CustomizeDiff: inlineAdapterIPCollisionDiff(rec.fn())}
			}
			ty := (&schema.Resource{Schema: sf.schema}).CoreConfigSchema().ImpliedType()
			const ownMAC = "aa:bb:cc:dd:ee:ff"
			// The recorded state of a Compute VM carries the MAC of each adapter
			// (Optional+Computed, filled by the read); the configuration does not.
			doc := func(withID bool) map[string]interface{} {
				adapter := map[string]interface{}{"network_id": "net-a", "ip_address": vmiStaticIP}
				d := map[string]interface{}{
					"name":               "vm",
					"os_network_adapter": []interface{}{adapter},
				}
				if withID {
					d["id"] = vmiSelfID
					adapter["mac_address"] = ownMAC
				}
				return d
			}

			t.Run("with an identity the hook consults IPAM and excuses the VM's own registration (machine link)", func(t *testing.T) {
				rec := &recordingConflict{holder: staticIPHeldBy(vmiSelfID, sf.source)}
				if _, err := planResource(t, newRes(rec), ctyFromJSON(t, ty, doc(true)), ctyFromJSON(t, ty, doc(false))); err != nil {
					t.Fatalf("own registration must pass with an identity: %v", err)
				}
				assertCalls(t, rec, "net-a|"+vmiStaticIP)
			})

			// The shape the client fixtures carry for xoa/vmware rows: no machine link,
			// ownership readable only from the MAC. The planned block's mac_address
			// (from the state) is the resource's positive proof that it is its own.
			t.Run("with an identity a MAC-owned registration with NO machine link is recognised as the VM's own", func(t *testing.T) {
				rec := &recordingConflict{holder: staticIPHeldByMAC("", sf.source, "AA:BB:CC:DD:EE:FF")} // case differs on purpose
				if _, err := planResource(t, newRes(rec), ctyFromJSON(t, ty, doc(true)), ctyFromJSON(t, ty, doc(false))); err != nil {
					t.Fatalf("a registration on this VM's own adapter MAC must not be a conflict: %v", err)
				}
				assertCalls(t, rec, "net-a|"+vmiStaticIP)
			})

			t.Run("with an identity a registration on a FOREIGN MAC with no machine link is refused", func(t *testing.T) {
				rec := &recordingConflict{holder: staticIPHeldByMAC("", sf.source, "de:ad:be:ef:00:01")}
				_, err := planResource(t, newRes(rec), ctyFromJSON(t, ty, doc(true)), ctyFromJSON(t, ty, doc(false)))
				if err == nil {
					t.Fatal("a MAC that is not one of this VM's adapters is not proof of ownership: the plan must refuse")
				}
				if !strings.Contains(err.Error(), "de:ad:be:ef:00:01") {
					t.Fatalf("the refusal must name the holder MAC: %v", err)
				}
				assertCalls(t, rec, "net-a|"+vmiStaticIP)
			})

			t.Run("with an identity another holder is refused", func(t *testing.T) {
				rec := &recordingConflict{holder: staticIPHeldBy("someone-else", sf.source)}
				if _, err := planResource(t, newRes(rec), ctyFromJSON(t, ty, doc(true)), ctyFromJSON(t, ty, doc(false))); err == nil {
					t.Fatal("a foreign holder must be refused when the plan carries an identity")
				}
				assertCalls(t, rec, "net-a|"+vmiStaticIP)
			})

			// The registration of a MAC-owned adapter may carry no VM object at all
			// (vpc_unit_test.go fixtures). Without an identity that shape, like every
			// other, is left to the pre-create check — the hook must not ask.
			t.Run("without an identity the hook defers, whatever the holder looks like", func(t *testing.T) {
				for _, holder := range []*client.StaticIP{
					staticIPHeldBy(vmiSelfID, sf.source),
					staticIPHeldBy("", sf.source), // MAC-owned, no VM link
					staticIPHeldBy("someone-else", sf.source),
				} {
					rec := &recordingConflict{holder: holder}
					if _, err := planResource(t, newRes(rec), cty.NullVal(ty), ctyFromJSON(t, ty, doc(false))); err != nil {
						t.Fatalf("no-identity plan must defer for holder %+v: %v", holder, err)
					}
					assertCalls(t, rec)
				}
			})
		})
	}
}
