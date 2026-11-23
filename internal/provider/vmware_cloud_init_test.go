package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-log/tflogtest"
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func cloudInitStateConfig() (*terraform.InstanceState, *terraform.ResourceConfig) {
	state := &terraform.InstanceState{ID: "vm-investigation", Attributes: map[string]string{
		"id": "vm-investigation", "name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
		"guest_operating_system_moref": "ubuntu64Guest", "memory": "2147483648", "cpu": "2", "num_cores_per_socket": "1",
		"power_state": "off", "cloud_init.%": "1", "cloud_init.user-data": "b2xk",
	}}
	cfg := terraform.NewResourceConfigRaw(map[string]interface{}{
		"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc", "guest_operating_system_moref": "ubuntu64Guest",
		"memory": 2147483648, "cpu": 2, "num_cores_per_socket": 1, "power_state": "off",
		"cloud_init": map[string]interface{}{"user-data": "bmV3"},
	})
	return state, cfg
}

// Exercise the SDK state boundary: the former update silently persisted the new
// map despite making no vApp request, permanently suppressing subsequent diffs.
func TestVMwareCloudInitApplyReplacesBeforeDeleting(t *testing.T) {
	r := resourceVirtualMachine()
	state, cfg := cloudInitStateConfig()
	props := []map[string]interface{}{
		{"key": float64(7000), "id": "user-data", "value": "b2xk", "label": "User data", "type": "string", "userConfigurable": float64(0), "classId": float64(0), "defaultValue": float64(0), "typeReference": float64(0), "description": "Preserve metadata"},
		{"key": float64(7001), "id": "sentinel", "value": "untouched", "type": "boolean"},
	}
	writes := []string{}
	c := newAssignTestClient(t, func(w http.ResponseWriter, q *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case q.Method == "GET" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation":
			json.NewEncoder(w).Encode(map[string]interface{}{"id": "vm-investigation", "name": "investigation", "powerState": "stopped", "vAppEnabled": true, "vAppProperties": props, "memory": 2147483648, "cpu": 2, "numCoresPerSocket": 1, "datacenter": map[string]string{"id": "dc"}, "hostCluster": map[string]string{"id": "hc"}, "operatingSystemMoref": "ubuntu64Guest"})
		case q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation/vapp_properties" && q.Method == "POST":
			writes = append(writes, q.Method)
			var added []map[string]interface{}
			if err := json.NewDecoder(q.Body).Decode(&added); err != nil {
				t.Fatal(err)
			}
			if len(props) != 2 || len(added) != 1 || added[0]["description"] != "Preserve metadata" {
				t.Fatal("original must survive until metadata clone confirmed")
			}
			added[0]["key"] = float64(7002)
			props = append(props, added[0])
			w.Header().Set("Location", "act-1")
			w.WriteHeader(201)
		case q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation/vapp_properties" && q.Method == "DELETE":
			writes = append(writes, q.Method)
			var req struct {
				Keys []int `json:"keys"`
			}
			json.NewDecoder(q.Body).Decode(&req)
			if len(props) != 3 || len(req.Keys) != 1 || req.Keys[0] != 7000 {
				t.Fatal("delete only original after replacement exists")
			}
			props = props[1:]
			w.Header().Set("Location", "act-1")
			w.WriteHeader(201)
		case q.Method == "GET" && q.URL.Path == "/activity/v1/activities/act-1":
			fmt.Fprint(w, `{"id":"act-1","state":{"completed":{}}}`)
		case q.Method == "GET" && (q.URL.Path == "/backup/v1/spp/policies" || q.URL.Path == "/tag/v1/tags/resources/vm-investigation"):
			fmt.Fprint(w, `[]`)
		default:
			t.Errorf("unexpected request %s %s", q.Method, q.URL.Path)
			w.WriteHeader(500)
		}
	})
	diff, err := r.Diff(context.Background(), state, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if diff.RequiresNew() {
		t.Fatal("cloud_init update must preserve VM")
	}
	next, diags := r.Apply(context.Background(), state, diff, c)
	if diags.HasError() {
		t.Fatal(diags)
	}
	if fmt.Sprint(writes) != "[POST DELETE]" || len(props) != 2 || props[1]["value"] != "bmV3" {
		t.Fatalf("cloud_init must reach VMware; writes=%v", writes)
	}
	if next.ID != state.ID || next.Attributes["cloud_init.user-data"] != "bmV3" {
		t.Fatal("incorrect persisted state")
	}
	after, err := r.Diff(context.Background(), next, cfg, nil)
	if err != nil {
		t.Fatal(err)
	}
	if after != nil && after.Attributes["cloud_init.user-data"] != nil {
		t.Fatal("confirmed update must converge")
	}
}

func TestVMwareCloudInitFailureRetainsSDKState(t *testing.T) {
	for _, mode := range []string{"read403", "read206", "missing-list", "null-list", "wrong-id", "duplicate-key", "duplicate-id", "running-no-optin", "power-unknown", "disabled", "missing-managed", "drift", "unsafe-value", "empty-value", "mixed-change-unknown", "post-lost", "post-delayed", "wait-add", "read-add", "wrong-clone", "collateral-change", "concurrent-before-delete", "delete-lost", "wait-delete", "read-after-delete", "final-vapp-check", "verified-then-read-fails", "cancel", "id-missing", "id-null", "id-number", "later-invalid"} {
		t.Run(mode, func(t *testing.T) {
			r := resourceVirtualMachine()
			state, cfg := cloudInitStateConfig()
			if strings.HasPrefix(mode, "id-") {
				cfg.Raw["cloud_init"] = map[string]interface{}{"user-data": "b2xk", "hostname": "new"}
				cfg.Config = cfg.Raw
			}
			if mode == "later-invalid" {
				state.Attributes["cloud_init.%"] = "2"
				state.Attributes["cloud_init.hostname"] = "before"
				cfg.Raw["cloud_init"] = map[string]interface{}{"user-data": "bmV3", "hostname": "after"}
				cfg.Config = cfg.Raw
			}
			if mode == "unsafe-value" {
				cfg.Raw["cloud_init"] = map[string]interface{}{"user-data": "unsafe/value"}
				cfg.Config = cfg.Raw
			}
			if mode == "empty-value" {
				cfg.Raw["cloud_init"] = map[string]interface{}{"user-data": ""}
				cfg.Config = cfg.Raw
			}
			// The ONE path that still reaches the apply-time sole-change check:
			// an attribute whose new value the plan could not resolve. The plan
			// guard deliberately fails open there, so without this check a
			// cloud_init update would be interleaved with a real change to
			// another attribute.
			if mode == "mixed-change-unknown" {
				cfg.Raw["name"] = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
				cfg.Config = cfg.Raw
			}
			if mode == "running-no-optin" {
				cfg.Raw["power_state"] = "on"
				cfg.Config = cfg.Raw
			}
			props := []map[string]interface{}{{"key": float64(7000), "id": "user-data", "value": "b2xk", "label": "User data", "type": "string", "userConfigurable": float64(1), "description": "preserve me"}}
			if mode == "duplicate-key" {
				props = append(props, map[string]interface{}{"key": float64(7000), "id": "sentinel"})
			}
			if mode == "duplicate-id" {
				props = append(props, map[string]interface{}{"key": float64(7001), "id": "user-data", "value": "other"})
			}
			if mode == "missing-managed" {
				props = []map[string]interface{}{}
			}
			if mode == "drift" {
				props[0]["value"] = "external"
			}
			if strings.HasPrefix(mode, "id-") {
				malformed := map[string]interface{}{"key": float64(7010)}
				if mode == "id-null" {
					malformed["id"] = nil
				}
				if mode == "id-number" {
					malformed["id"] = 42
				}
				props = append(props, malformed)
			}
			if mode == "later-invalid" {
				props[0]["description"] = "not/replayable"
				props = append(props, map[string]interface{}{"key": float64(7010), "id": "hostname", "value": "before", "label": "Hostname", "type": "string"})
			}
			writes := []string{}
			readsAfterPost := 0
			readsAfterDelete := 0
			var pending map[string]interface{}
			var logs bytes.Buffer
			ctx, cancel := context.WithCancel(tflogtest.RootLogger(context.Background(), &logs))
			defer cancel()
			c := newAssignTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if q.Method == "GET" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation" {
					if len(writes) == 1 {
						readsAfterPost++
					}
					if len(writes) == 2 {
						readsAfterDelete++
					}
					if mode == "read403" || mode == "read-add" && len(writes) == 1 || mode == "read-after-delete" && len(writes) == 2 || mode == "final-vapp-check" && readsAfterDelete == 2 || mode == "verified-then-read-fails" && readsAfterDelete == 3 {
						w.WriteHeader(403)
						return
					}
					if mode == "read206" {
						w.WriteHeader(206)
					}
					vm := map[string]interface{}{"id": "vm-investigation", "name": "investigation", "powerState": "stopped", "vAppEnabled": true, "vAppProperties": props, "memory": 2147483648, "cpu": 2, "numCoresPerSocket": 1, "datacenter": map[string]string{"id": "dc"}, "hostCluster": map[string]string{"id": "hc"}, "operatingSystemMoref": "ubuntu64Guest"}
					switch mode {
					case "missing-list":
						delete(vm, "vAppProperties")
					case "null-list":
						vm["vAppProperties"] = nil
					case "wrong-id":
						vm["id"] = "another-vm"
					case "running-no-optin":
						// Live authority: the prior state says "off" (stale), so the
						// plan-time guard cannot fire. Only an apply that reads the
						// LIVE power state refuses this.
						vm["powerState"] = "running"
					case "power-unknown":
						// Not "running" is not evidence of "stopped": fail closed.
						vm["powerState"] = "suspended"
					case "disabled":
						vm["vAppEnabled"] = false
					}
					if mode == "concurrent-before-delete" && readsAfterPost == 2 {
						props[0]["label"] = "concurrent change"
					}
					json.NewEncoder(w).Encode(vm)
					return
				}
				if q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation/vapp_properties" {
					writes = append(writes, q.Method)
					if q.Method == "POST" {
						var added []map[string]interface{}
						json.NewDecoder(q.Body).Decode(&added)
						added[0]["key"] = float64(7001)
						if mode == "post-delayed" {
							pending = added[0]
							w.WriteHeader(500)
							return
						}
						props = append(props, added[0])
						if mode == "post-lost" {
							w.WriteHeader(500)
							fmt.Fprint(w, "secret-from-api")
							return
						}
						if mode == "wrong-clone" {
							props[1]["value"] = "normalized"
						}
						if mode == "collateral-change" {
							props[0]["label"] = "changed unexpectedly"
						}
						if mode == "cancel" {
							cancel()
						}
					} else if q.Method == "DELETE" {
						props = props[1:]
						if mode == "delete-lost" {
							w.WriteHeader(500)
							return
						}
					} else {
						t.Errorf("unexpected mutation %s", q.Method)
					}
					w.Header().Set("Location", "act-1")
					w.WriteHeader(201)
					return
				}
				if q.URL.Path == "/activity/v1/activities/act-1" {
					if mode == "wait-add" && len(writes) == 1 || mode == "wait-delete" && len(writes) == 2 {
						fmt.Fprint(w, `{"id":"act-1","state":{"failed":{"reason":"secret-from-api"}}}`)
					} else {
						fmt.Fprint(w, `{"id":"act-1","state":{"completed":{}}}`)
					}
					return
				}
				if q.Method == "GET" && (q.URL.Path == "/backup/v1/spp/policies" || q.URL.Path == "/tag/v1/tags/resources/vm-investigation") {
					fmt.Fprint(w, `[]`)
					return
				}
				t.Errorf("unexpected request %s %s", q.Method, q.URL.Path)
				w.WriteHeader(500)
			})
			diff, err := r.Diff(ctx, state, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			next, diags := r.Apply(ctx, state, diff, c)
			if !diags.HasError() {
				t.Fatal("expected failure")
			}
			// The vApp transaction is verified before the general VM read, so a
			// failure in that read must NOT give the prior intent back: the
			// machine already holds the new values, and restoring the old ones
			// would be permanent — the refresh never adds a key back and never
			// removes one it can no longer observe.
			verified := mode == "verified-then-read-fails"
			wantIntent := "b2xk"
			if verified {
				wantIntent = "bmV3"
			}
			if next.ID != state.ID || next.Attributes["cloud_init.user-data"] != wantIntent {
				t.Fatalf("persisted cloud_init = %q, want %q", next.Attributes["cloud_init.user-data"], wantIntent)
			}
			again, err := r.Diff(context.Background(), next, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			retryKey := "cloud_init.user-data"
			if strings.HasPrefix(mode, "id-") {
				retryKey = "cloud_init.hostname"
			}
			if verified {
				// ...and because it was kept, the next plan converges instead of
				// replaying a transaction the machine has already accepted.
				if again != nil && again.Attributes[retryKey] != nil {
					t.Fatal("a verified transaction must converge, not replay")
				}
			} else if again == nil || again.Attributes[retryKey] == nil {
				t.Fatal("failed update suppressed retry intent")
			}
			expectedWrites := 0
			switch mode {
			case "post-lost", "post-delayed", "wait-add", "read-add", "wrong-clone", "collateral-change", "concurrent-before-delete", "cancel":
				expectedWrites = 1
			case "delete-lost", "wait-delete", "read-after-delete", "final-vapp-check", "verified-then-read-fails":
				expectedWrites = 2
			}
			if len(writes) != expectedWrites {
				t.Fatalf("unexpected mutation count %v, wanted %d", writes, expectedWrites)
			}
			message := fmt.Sprint(diags)
			if strings.Contains(message+logs.String(), "secret-from-api") {
				t.Fatal("diagnostic leaked payload")
			}
			// Asserting "it failed" is not enough when several guards could have
			// produced the failure: pin each power refusal to its OWN diagnostic,
			// or removing it passes anyway because a later check catches the case.
			if want := map[string]string{
				"power-unknown":        "unsupported power state",
				"running-no-optin":     "allow_vm_restart",
				"mixed-change-unknown": `"name"`,
			}[mode]; want != "" && !strings.Contains(message, want) {
				t.Fatalf("mode %s must be refused by its own guard; got: %s", mode, message)
			}
			if got := strings.Contains(message, "Do not reapply"); got != (expectedWrites > 0 && !verified) {
				t.Fatalf("recovery alarm = %v: it must fire for an uncertain mutation and NOT for a verified transaction", got)
			}
			if mode == "post-delayed" {
				props = append(props, pending)
			} // backend completes after failed apply
			if mode == "post-lost" || mode == "post-delayed" || mode == "wait-add" {
				// After terminal settlement, duplicates still require manual reconciliation;
				// a second invocation must refuse before issuing another mutation.
				_, second := r.Apply(context.Background(), next, again, c)
				if !second.HasError() || len(writes) != 1 {
					t.Fatal("retry replayed uncertain addition")
				}
				if props[0]["description"] != "preserve me" {
					t.Fatal("original metadata was destroyed")
				}
			}
		})
	}
}

func TestVMwareCloudInitPlanOwnershipAndValidation(t *testing.T) {
	original := map[string]interface{}{"key": float64(1), "id": "hostname", "value": "before", "label": "Hostname", "type": "string", "userConfigurable": float64(1), "description": float64(0), "typeReference": float64(0), "classId": float64(0)}
	snapshot := func() *client.VAppSnapshot {
		return &client.VAppSnapshot{ID: "vm", PowerState: "stopped", Enabled: true, Properties: []map[string]interface{}{original}}
	}
	old := map[string]interface{}{"hostname": "before"}
	t.Run("remove only previously managed key", func(t *testing.T) {
		edits, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{}, nil)
		if err != nil || len(edits) != 1 || edits[0].replacement != nil || !reflect.DeepEqual(edits[0].original, original) {
			t.Fatalf("bad removal plan: %v", err)
		}
	})
	t.Run("add only absent key", func(t *testing.T) {
		edits, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{"hostname": "before", "new": "value"}, nil)
		if err != nil || len(edits) != 1 || edits[0].original != nil || edits[0].replacement["id"] != "new" {
			t.Fatalf("bad addition plan: %v", err)
		}
	})
	// This pair replaces a test that asserted the opposite. Refusing to adopt
	// made an IMPORTED machine unmanageable: an import records no cloud_init, so
	// every declared key is unmanaged and every one already exists on the
	// machine, and no configuration could converge — not even one declaring the
	// values the machine already held. Terraform is the source of truth, so the
	// declared value wins.
	t.Run("adopt a key whose live value already matches", func(t *testing.T) {
		edits, err := planCloudInitEdits(snapshot(), nil, map[string]interface{}{"hostname": "before"}, nil)
		if err != nil {
			t.Fatalf("declaring the value the machine already holds must converge: %v", err)
		}
		if len(edits) != 0 {
			t.Fatalf("a value already in place must plan no write; got %d edit(s)", len(edits))
		}
	})
	t.Run("adopt a key by overwriting its live value", func(t *testing.T) {
		edits, err := planCloudInitEdits(snapshot(), nil, map[string]interface{}{"hostname": "after"}, nil)
		if err != nil {
			t.Fatalf("the declared value must win over the live one: %v", err)
		}
		if len(edits) != 1 || edits[0].replacement == nil {
			t.Fatalf("adoption must plan a replacement; got %#v", edits)
		}
		if !reflect.DeepEqual(edits[0].original, original) {
			t.Fatal("adoption must replace the LIVE descriptor, not invent a new one")
		}
		if edits[0].replacement["value"] != "after" {
			t.Fatalf("replacement value = %v, want \"after\"", edits[0].replacement["value"])
		}
		// The point of cloning the live descriptor: metadata the image provided
		// survives the add-then-delete, which a fresh property would discard.
		if edits[0].replacement["label"] != "Hostname" {
			t.Fatalf("adoption dropped the live metadata: %#v", edits[0].replacement)
		}
	})
	// Adoption must not weaken the guard that protects a key already tracked.
	t.Run("a tracked key that drifted is still refused", func(t *testing.T) {
		vm := snapshot()
		vm.Properties = []map[string]interface{}{{"key": float64(1), "id": "hostname", "value": "changed-elsewhere"}}
		_, err := planCloudInitEdits(vm, old, map[string]interface{}{"hostname": "after"}, nil)
		if err == nil || !strings.Contains(err.Error(), "drifted") {
			t.Fatalf("a drifted tracked key must still be refused; got: %v", err)
		}
	})
	// M1, review of 23/11. A value the platform DEFORMED on the way in must not
	// be reported as drift, or the upgrade to 1.13.0 is a dead end for every
	// machine created before it whose cloud_init carried a '/': state holds the
	// declared form, the vApp holds the escaped one, keeping the declaration
	// fails validate, and both ways out of that — removing the key, or giving it
	// a transmittable value — are CHANGES, which the drift guard refused. The
	// only exit was `terraform state rm` + `terraform import`.
	deformed := func(live string) *client.VAppSnapshot {
		p := map[string]interface{}{}
		for k, v := range original {
			p[k] = v
		}
		p["value"] = live
		return &client.VAppSnapshot{ID: "vm", PowerState: "stopped", Enabled: true, Properties: []map[string]interface{}{p}}
	}
	escapedIntent := map[string]interface{}{"hostname": "a/b"}
	t.Run("a value the platform escaped can be removed", func(t *testing.T) {
		edits, err := planCloudInitEdits(deformed("a&#x2F;b"), escapedIntent, map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("removing a key the platform deformed must be allowed: %v", err)
		}
		if len(edits) != 1 || edits[0].replacement != nil {
			t.Fatalf("want exactly one deletion; got %#v", edits)
		}
	})
	t.Run("a value the platform escaped can be replaced by a transmittable one", func(t *testing.T) {
		edits, err := planCloudInitEdits(deformed("a&#x2F;b"), escapedIntent,
			map[string]interface{}{"hostname": "safe"}, nil)
		if err != nil {
			t.Fatalf("repairing a key the platform deformed must be allowed: %v", err)
		}
		if len(edits) != 1 || edits[0].replacement == nil {
			t.Fatalf("want exactly one replacement; got %#v", edits)
		}
		if edits[0].replacement["value"] != "safe" {
			t.Fatalf("replacement value = %v, want \"safe\"", edits[0].replacement["value"])
		}
		// Still a replacement of the LIVE descriptor: the image's metadata must
		// survive the repair, as it does for any other adoption.
		if edits[0].replacement["label"] != "Hostname" {
			t.Fatalf("the repair dropped the live metadata: %#v", edits[0].replacement)
		}
	})
	t.Run("a value the platform trimmed is not drift", func(t *testing.T) {
		_, err := planCloudInitEdits(deformed("web01"), map[string]interface{}{"hostname": " web01 "},
			map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("a trimmed value is the platform's own storage of our intent: %v", err)
		}
	})
	// The three cases that keep the exemption NARROW. It recognises this
	// provider's recorded intent in the form the platform was MEASURED to store
	// it, and nothing else: a wrong entity form here would silently classify
	// somebody else's edit as "our value, escaped" and suppress real drift.
	t.Run("a live value that is not the escape of the recorded one is still drift", func(t *testing.T) {
		_, err := planCloudInitEdits(deformed("a&#x2F;c"), escapedIntent, map[string]interface{}{}, nil)
		if err == nil || !strings.Contains(err.Error(), "drifted") {
			t.Fatalf("a different value must still be refused; got: %v", err)
		}
	})
	t.Run("an unmeasured entity form is still drift", func(t *testing.T) {
		// The table holds the forms that were MEASURED. "&#38;" is the decimal
		// entity for '&' — a perfectly plausible escaping that this platform was
		// NOT observed to produce. Recognising it would widen the exemption on
		// an assumption, which is the one thing a recognition must not do.
		_, err := planCloudInitEdits(deformed("a&#38;b"), map[string]interface{}{"hostname": "a&b"},
			map[string]interface{}{}, nil)
		if err == nil || !strings.Contains(err.Error(), "drifted") {
			t.Fatalf("an unmeasured entity form must still be refused; got: %v", err)
		}
	})
	// The characters the DEPLOYMENT path was measured escaping differ from the
	// ones the vApp path escapes: on one environment the deployment escaped '<'
	// alone, on another it escaped '/'. A live run proved that reproducing a
	// single forward form leaves the dead end wide open on the first of those,
	// so each measured character has a case, and so does a partial subset.
	t.Run("a '<' the deployment path escaped is not drift", func(t *testing.T) {
		_, err := planCloudInitEdits(deformed("a&lt;b"), map[string]interface{}{"hostname": "a<b"},
			map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("the escape this platform actually produces must be recognised: %v", err)
		}
	})
	t.Run("a partial subset is not drift", func(t *testing.T) {
		// Verbatim from the live measurement: the deployment path escaped '<'
		// and left the other seven characters alone, in the same value.
		_, err := planCloudInitEdits(deformed(`A/B&C&lt;D>E"F'G`+"`"+`H\I`),
			map[string]interface{}{"hostname": `A/B&C<D>E"F'G` + "`" + `H\I`},
			map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("a value escaped in part must be recognised: %v", err)
		}
	})
	t.Run("the whole measured table is not drift", func(t *testing.T) {
		// Verbatim from the live measurement of the vApp property path.
		_, err := planCloudInitEdits(
			deformed("A&#x2F;B&amp;C&lt;D&gt;E&quot;F&#x27;G&#96;H&#x5C;I"),
			map[string]interface{}{"hostname": `A/B&C<D>E"F'G` + "`" + `H\I`},
			map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("the full measured escaping must be recognised: %v", err)
		}
	})
	t.Run("a literal entity in the declared value is not mangled", func(t *testing.T) {
		// The ordering hazard. A declared value containing a literal "&lt;" is
		// stored as "&amp;lt;". Un-escaping '&' before the others turns that
		// into "<" — a character the operator never wrote — and the comparison
		// then fails, so the operator is told their own value has drifted.
		_, err := planCloudInitEdits(deformed("a&amp;lt;b"), map[string]interface{}{"hostname": "a&lt;b"},
			map[string]interface{}{}, nil)
		if err != nil {
			t.Fatalf("a literal entity must survive the reversal: %v", err)
		}
	})
	t.Run("the escaped form is recognised only against the key it belongs to", func(t *testing.T) {
		// The live value is a valid escape — of a value this key never held.
		_, err := planCloudInitEdits(deformed("a&#x2F;b"), map[string]interface{}{"hostname": "c/d"},
			map[string]interface{}{}, nil)
		if err == nil || !strings.Contains(err.Error(), "drifted") {
			t.Fatalf("an escape of another value must still be refused; got: %v", err)
		}
	})

	// Resolving a collision must have a way out. Removing the key from
	// deploy_options is not one: that attribute is ForceNew, so editing it
	// replaces the machine. The only in-place remedy is to drop the key from
	// cloud_init — which is a CHANGE, so the refusal below would catch it and
	// leave the operator with no non-destructive move at all.
	//
	// Dropping it must also not DELETE the property: deploy_options still
	// declares it, and its value is the one the machine actually holds.
	t.Run("stop managing a key deploy_options owns", func(t *testing.T) {
		edits, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{},
			map[string]interface{}{"hostname": "legacy"})
		if err != nil {
			t.Fatalf("dropping a key deploy_options owns must be allowed: %v", err)
		}
		if len(edits) != 0 {
			t.Fatalf("a key deploy_options owns must be neither written nor deleted; got %d edit(s)", len(edits))
		}
	})
	// The LAST barrier before a write. The plan-time guard refuses the whole
	// overlap earlier, so Terraform no longer reaches this one — it stays
	// because nothing may write a shared key, whatever path got here.
	t.Run("refuse a key also owned by deploy_options", func(t *testing.T) {
		_, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{"hostname": "after"},
			map[string]interface{}{"hostname": "legacy"})
		if err == nil {
			t.Fatal("planned a write for a key deploy_options also declares")
		}
		if !strings.Contains(err.Error(), "deploy_options") {
			t.Fatalf("the refusal must name deploy_options; got: %v", err)
		}
	})
	t.Run("settled already desired", func(t *testing.T) {
		vm := snapshot()
		vm.Properties = []map[string]interface{}{{"key": float64(1), "id": "hostname", "value": "after"}}
		edits, err := planCloudInitEdits(vm, old, map[string]interface{}{"hostname": "after"}, nil)
		if err != nil || len(edits) != 0 {
			t.Fatal("settled desired value must be no-op")
		}
	})
	for _, value := range []string{"", " foo", "foo\n", "x/y", "a&b", "<tag>", "a\"b", "a'b", "a`b", "a\\b"} {
		t.Run("reject normalized "+value, func(t *testing.T) {
			_, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{"hostname": value}, nil)
			if err == nil {
				t.Fatal("unsafe value accepted")
			}
		})
	}
	for _, id := range []string{"", "a/b", strings.Repeat("x", 61)} {
		t.Run("invalid id", func(t *testing.T) {
			_, err := planCloudInitEdits(snapshot(), old, map[string]interface{}{"hostname": "before", id: "new"}, nil)
			if err == nil {
				t.Fatal("unsafe id accepted")
			}
		})
	}
	for _, field := range []string{"description", "typeReference", "unknown"} {
		t.Run("unreplayable metadata "+field, func(t *testing.T) {
			vm := snapshot()
			p := map[string]interface{}{}
			for k, v := range original {
				p[k] = v
			}
			p[field] = "unsafe/value"
			vm.Properties = []map[string]interface{}{p}
			_, err := planCloudInitEdits(vm, old, map[string]interface{}{"hostname": "after"}, nil)
			if err == nil {
				t.Fatal("unsafe metadata replayed")
			}
		})
	}
}

func TestVMwareCloudInitApplyAddRemoveAndPartialBatch(t *testing.T) {
	for _, mode := range []string{"add", "remove", "partial-batch", "unchanged"} {
		t.Run(mode, func(t *testing.T) {
			r := resourceVirtualMachine()
			state, _ := cloudInitStateConfig()
			delete(state.Attributes, "cloud_init.user-data")
			state.Attributes["cloud_init.hostname"] = "before"
			desired := map[string]interface{}{}
			switch mode {
			case "add":
				desired = map[string]interface{}{"hostname": "before", "user-data": "new"}
			case "partial-batch":
				desired = map[string]interface{}{"hostname": "after", "user-data": "new"}
			case "unchanged":
				desired = map[string]interface{}{"hostname": "before"}
			}
			cfg := terraform.NewResourceConfigRaw(map[string]interface{}{"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc", "guest_operating_system_moref": "ubuntu64Guest", "memory": 2147483648, "cpu": 2, "num_cores_per_socket": 1, "power_state": "off", "cloud_init": desired})
			sentinel := map[string]interface{}{"key": float64(9000), "id": "unmanaged", "value": true, "type": "boolean", "custom": map[string]interface{}{"keep": true}}
			props := []map[string]interface{}{{"key": float64(7000), "id": "hostname", "value": "before", "label": "User data", "type": "string", "userConfigurable": float64(1)}, sentinel}
			writes := []string{}
			c := newAssignTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation" && q.Method == "GET" {
					json.NewEncoder(w).Encode(map[string]interface{}{"id": "vm-investigation", "name": "investigation", "powerState": "stopped", "vAppEnabled": mode != "unchanged", "vAppProperties": props, "memory": 2147483648, "cpu": 2, "numCoresPerSocket": 1, "datacenter": map[string]string{"id": "dc"}, "hostCluster": map[string]string{"id": "hc"}, "operatingSystemMoref": "ubuntu64Guest"})
					return
				}
				if q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation/vapp_properties" {
					writes = append(writes, q.Method)
					if mode == "partial-batch" && len(writes) == 3 {
						w.WriteHeader(500)
						return
					}
					switch q.Method {
					case "POST":
						var added []map[string]interface{}
						json.NewDecoder(q.Body).Decode(&added)
						added[0]["key"] = float64(9001)
						props = append(props, added[0])
					case "DELETE":
						var req struct {
							Keys []int `json:"keys"`
						}
						json.NewDecoder(q.Body).Decode(&req)
						if len(req.Keys) != 1 || req.Keys[0] != 7000 {
							t.Error("deleted unrelated property")
						}
						props = props[1:]
					default:
						t.Error("unexpected mutation")
					}
					w.Header().Set("Location", "act-1")
					w.WriteHeader(201)
					return
				}
				if q.URL.Path == "/activity/v1/activities/act-1" {
					fmt.Fprint(w, `{"id":"act-1","state":{"completed":{}}}`)
					return
				}
				if q.Method == "GET" && (q.URL.Path == "/backup/v1/spp/policies" || q.URL.Path == "/tag/v1/tags/resources/vm-investigation") {
					fmt.Fprint(w, `[]`)
					return
				}
				t.Errorf("unexpected request %s %s", q.Method, q.URL.Path)
				w.WriteHeader(500)
			})
			diff, err := r.Diff(context.Background(), state, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if diff != nil && diff.RequiresNew() {
				t.Fatal("replacement requested")
			}
			next, diags := r.Apply(context.Background(), state, diff, c)
			if mode == "partial-batch" {
				if !diags.HasError() || next.Attributes["cloud_init.hostname"] != "before" || next.Attributes["cloud_init.user-data"] != "" || fmt.Sprint(writes) != "[POST DELETE POST]" {
					t.Fatal("partial batch was not stopped with prior intent retained")
				}
				if props[1]["value"] != "after" {
					t.Fatal("test must exercise real partial backend change")
				}
			} else {
				if diags.HasError() {
					t.Fatal(diags)
				}
				expected := map[string]string{"add": "[POST]", "remove": "[DELETE]", "unchanged": "[]"}[mode]
				if fmt.Sprint(writes) != expected {
					t.Fatalf("unexpected writes: %v", writes)
				}
				for k, v := range desired {
					if next.Attributes["cloud_init."+k] != v {
						t.Fatal("desired state not persisted")
					}
				}
				if mode == "remove" && next.Attributes["cloud_init.hostname"] != "" {
					t.Fatal("removed property retained in state")
				}
			}
			if next.ID != state.ID {
				t.Fatal("identity changed")
			}
			found := false
			for _, p := range props {
				if p["key"] == float64(9000) {
					found = reflect.DeepEqual(p, sentinel)
				}
			}
			if !found {
				t.Fatal("unmanaged descriptor changed")
			}
		})
	}
}

// cloudInitPowerFake is a stateful VMware fake for the #549 power-cycle path.
//
// Its live power state changes ONLY through a power operation the fake itself
// accepted. That is what keeps the assertions below from being complacent: a
// provider that forgets to power the machine off is caught because every vApp
// verification asserts "stopped" and the fake keeps reporting "running"; one
// that forgets to power it back on leaves the fake reporting "stopped"; and one
// that powers it on after a failed write is caught by the same field.
type cloudInitPowerFake struct {
	t         *testing.T
	power     string
	props     []map[string]interface{}
	actions   []string
	failPower string // power action whose ACTIVITY fails
	failCall  string // vApp method whose HTTP CALL fails outright
	failWrite string // vApp method whose ACTIVITY fails
	failRead  bool   // 403 the first read taken after the shutdown (once)
	staysOn   bool   // accept the power-off but keep reporting "running"
	onNoOp    bool   // accept the power-on but keep reporting "stopped"
	drift     bool   // mutate a descriptor on the first read after the shutdown
	oddLater  bool   // report an unsupported power state at settlement time
	dupSettle bool   // another writer adds a duplicate id before settlement
	modSettle bool   // another writer edits a descriptor before settlement
	dupAfter  bool   // another writer adds a duplicate id after the shutdown
	oddRestor bool   // the machine turns transitional before compensation
	extStart  bool   // another actor starts the machine while it is down
	failGetNo int    // 403 the Nth GET on the virtual machine endpoint (1-based)
	// The two response shapes the general read tolerates and the vApp
	// snapshot does not, each applied to the Nth GET (1-based).
	code206No   int
	nullPropsNo int
	gets        int  // GETs taken on the virtual machine endpoint
	reads       int  // VM reads taken since the last vApp write
	postOff     int  // reads taken since the shutdown
	secretLoc   bool // return an activity id carrying the secret marker
}

func (f *cloudInitPowerFake) handler(w http.ResponseWriter, q *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	afterShutdown := len(f.actions) == 1 && f.actions[0] == "power:off"
	switch {
	case q.Method == "GET" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation":
		f.gets++
		if f.failGetNo == f.gets {
			w.WriteHeader(403)
			return
		}
		props := f.props
		if f.nullPropsNo == f.gets {
			props = nil // encodes as JSON null, which the general read tolerates
		}
		if f.code206No == f.gets {
			w.WriteHeader(206)
		}
		if f.failRead && afterShutdown {
			f.failRead = false
			w.WriteHeader(403)
			return
		}
		if f.drift && afterShutdown {
			f.drift = false
			f.props[0]["label"] = "changed by someone else"
		}
		power := f.power
		if afterShutdown {
			f.postOff++
			if f.dupAfter && f.postOff == 1 {
				// Another writer slips a second descriptor carrying the same
				// textual id under us while the machine is down.
				f.props = append(f.props, map[string]interface{}{"key": float64(7900),
					"id": "user-data", "value": "written-by-someone-else", "label": "User data", "type": "string"})
			}
			if f.oddRestor && f.postOff > 1 {
				power = "suspending"
			}
			if f.extStart && f.postOff > 1 {
				// Somebody else powered it back on behind the provider's back.
				power = "running"
				f.power = "running"
			}
		}
		if n := len(f.actions); n > 0 && f.actions[n-1] == "DELETE" {
			f.reads++
			// Reads after the last write are: verify the deletion, the final vApp
			// verification, the general VM read — and only then settlement.
			if f.oddLater && f.reads > 3 {
				power = "suspending" // transitional: evidence of nothing
			}
			if f.modSettle && f.reads == 4 {
				f.props[len(f.props)-1]["label"] = "retitled by someone else"
			}
			if f.dupSettle && f.reads == 4 {
				f.props = append(f.props, map[string]interface{}{"key": float64(7900),
					"id": "user-data", "value": "written-by-someone-else", "label": "User data", "type": "string"})
			}
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "vm-investigation", "name": "investigation", "powerState": power,
			"vAppEnabled": true, "vAppProperties": props,
			"memory": 2147483648, "cpu": 2, "numCoresPerSocket": 1,
			"datacenter": map[string]string{"id": "dc"}, "hostCluster": map[string]string{"id": "hc"},
			"operatingSystemMoref": "ubuntu64Guest",
		})
	case q.Method == "GET" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/power/recommendations":
		fmt.Fprint(w, `[]`)
	case q.Method == "PATCH" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/power":
		var req struct {
			PowerAction string `json:"powerAction"`
		}
		json.NewDecoder(q.Body).Decode(&req)
		f.actions = append(f.actions, "power:"+req.PowerAction)
		if f.failPower == req.PowerAction {
			w.Header().Set("Location", "act-bad")
			w.WriteHeader(201)
			return
		}
		switch req.PowerAction {
		case "off":
			if !f.staysOn { // an accepted request whose effect has not settled
				f.power = "stopped"
			}
		case "on":
			if !f.onNoOp {
				f.power = "running"
			}
		default:
			f.t.Errorf("unexpected power action %q", req.PowerAction)
		}
		w.Header().Set("Location", "act-ok")
		w.WriteHeader(201)
	case q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation/vapp_properties":
		f.actions = append(f.actions, q.Method)
		if f.failCall == q.Method { // the call itself fails: the write may still have landed
			w.WriteHeader(500)
			return
		}
		if q.Method == "POST" {
			var added []map[string]interface{}
			json.NewDecoder(q.Body).Decode(&added)
			added[0]["key"] = float64(7001)
			f.props = append(f.props, added[0])
		} else {
			var req struct {
				Keys []int `json:"keys"`
			}
			json.NewDecoder(q.Body).Decode(&req)
			kept := f.props[:0]
			for _, p := range f.props {
				if len(req.Keys) != 1 || p["key"] != float64(req.Keys[0]) {
					kept = append(kept, p)
				}
			}
			if len(kept) == len(f.props) {
				f.t.Errorf("DELETE asked for keys %v, which match no property", req.Keys)
			}
			f.props = kept
		}
		if f.secretLoc {
			// An unconstrained Location header: the activity id must never be
			// echoed verbatim into a diagnostic.
			w.Header().Set("Location", "secret-from-api!x")
			w.WriteHeader(201)
			return
		}
		if f.failWrite == q.Method {
			w.Header().Set("Location", "act-bad")
			w.WriteHeader(201)
			return
		}
		w.Header().Set("Location", "act-ok")
		w.WriteHeader(201)
	case q.URL.Path == "/activity/v1/activities/act-ok":
		fmt.Fprint(w, `{"id":"act-ok","state":{"completed":{}}}`)
	case q.URL.Path == "/activity/v1/activities/act-bad",
		q.URL.Path == "/activity/v1/activities/secret-from-api!x":
		fmt.Fprint(w, `{"id":"act-bad","state":{"failed":{"reason":"secret-from-api"}}}`)
	case q.Method == "GET" && (q.URL.Path == "/backup/v1/spp/policies" || q.URL.Path == "/tag/v1/tags/resources/vm-investigation"):
		fmt.Fprint(w, `[]`)
	default:
		f.t.Errorf("unexpected request %s %s", q.Method, q.URL.Path)
		w.WriteHeader(500)
	}
}

// TestVMwareCloudInitPowerCycle pins the #549 power contract.
//
// Two invariants carry most of the weight, and each has a case AND its mirror
// image, because asserting only one of the two lets the opposite bug through:
//
//   - after a vApp write was ATTEMPTED the machine stays stopped (a lost
//     response still mutates the vApp, so it may hold a duplicate or a partial
//     edit — what cloudInitRecovery tells the operator not to boot into), while
//     a failure that provably wrote nothing gives the machine back;
//   - a transaction that was VERIFIED keeps its new cloud_init in state even if
//     the power settlement then fails, while an unverified one gives the prior
//     intent back. Rolling a verified transaction back would be permanent: the
//     read never hydrates cloud_init, so the next apply would refuse an added
//     key as unadopted, or a removed one as lost, forever.
func TestVMwareCloudInitPowerCycle(t *testing.T) {
	for _, tc := range []struct {
		name       string
		statePower string
		cfgPower   string
		allow      bool
		livePower  string
		liveValue  string // live user-data; equal to the desired value means zero edits
		otherAttr  string // a non-power attribute changed in the same apply
		unsafe     bool   // a configuration the planner can never apply
		failPower  string
		failCall   string
		failWrite  string
		failRead   bool
		staysOn    bool
		onNoOp     bool
		drift      bool
		oddLater   bool
		secretLoc  bool
		dupSettle  bool
		modSettle  bool
		dupAfter   bool
		oddRestor  bool
		extStart   bool
		planFails  bool
		applyFails bool
		actions    string
		wantLive   string
		wantState  string
		wantIntent string // cloud_init.user-data expected in the persisted state
		diagnostic string // a substring that pins the refusal to its OWN guard
		recovery   bool
	}{
		// --- successful paths ------------------------------------------------
		// allow_vm_restart is absent from the prior state and true in the
		// configuration, so this also proves the opt-in can be GRANTED in the
		// same apply as the change it authorises.
		{name: "running-optin-restarts", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			actions: "[power:off POST DELETE power:on]", wantLive: "running", wantState: "on", wantIntent: "bmV3"},
		// power_state = "off" is its own authorisation: nothing is interrupted
		// that the operator did not already ask to stop.
		{name: "target-off-needs-no-optin", statePower: "on", cfgPower: "off", livePower: "running",
			actions: "[power:off POST DELETE]", wantLive: "stopped", wantState: "off", wantIntent: "bmV3"},
		// The hole a poweredOff-gated restart would leave: already stopped, asked
		// to end running. Starting a stopped machine interrupts nothing.
		{name: "stopped-target-on-starts", statePower: "off", cfgPower: "on", livePower: "stopped",
			actions: "[POST DELETE power:on]", wantLive: "running", wantState: "on", wantIntent: "bmV3"},
		{name: "stopped-target-off-untouched", statePower: "off", cfgPower: "off", livePower: "stopped",
			actions: "[POST DELETE]", wantLive: "stopped", wantState: "off", wantIntent: "bmV3"},
		// A no-op must not cost an interruption, even on a running machine with
		// the opt-in granted. (The plan guard cannot know the live value already
		// matches — it has no live evidence — so it conservatively asks for the
		// opt-in first; the apply then discovers there is nothing to write.)
		{name: "noop-never-interrupts", statePower: "on", cfgPower: "on", allow: true, livePower: "running", liveValue: "bmV3",
			actions: "[]", wantLive: "running", wantState: "on", wantIntent: "bmV3"},
		// ...but the declared power state is still settled, in both directions.
		{name: "noop-still-settles-power-off", statePower: "on", cfgPower: "off", livePower: "running", liveValue: "bmV3",
			actions: "[power:off]", wantLive: "stopped", wantState: "off", wantIntent: "bmV3"},

		// --- refusals, each pinned to its own diagnostic ----------------------
		{name: "plan-refuses-without-optin", statePower: "on", cfgPower: "on", livePower: "running",
			planFails: true, diagnostic: "allow_vm_restart"},
		// Prior state is a stale proxy; only the live machine is authority.
		{name: "apply-refuses-on-live-authority", wantState: "off", statePower: "off", cfgPower: "on", livePower: "running",
			applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk", diagnostic: "allow_vm_restart"},
		// An unsupported live power state is not evidence of "stopped".
		{name: "unsupported-power-state-fails-closed", wantState: "off", statePower: "off", cfgPower: "off", livePower: "secret-from-api",
			applyFails: true, actions: "[]", wantIntent: "b2xk", wantLive: "secret-from-api",
			diagnostic: "unsupported power state"},
		// Exempting the two power attributes must not have opened the guard.
		{name: "other-attribute-still-refused", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			otherAttr: "name", planFails: true, diagnostic: `"name"`},
		{name: "other-attribute-tags-still-refused", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			otherAttr: "tags", planFails: true, diagnostic: `"tags"`},
		// An inapplicable configuration must cost NO downtime.
		{name: "invalid-config-refused-before-downtime", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			unsafe: true, applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "cannot update cloud_init"},

		// --- failures around the shutdown -------------------------------------
		// An unconfirmed shutdown must NOT be "compensated" by a power-on that
		// could race a shutdown still settling.
		{name: "unconfirmed-shutdown-fails-closed", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failRead: true, applyFails: true, actions: "[power:off]", wantLive: "stopped", wantIntent: "b2xk",
			diagnostic: "could not be re-read"},
		{name: "shutdown-reported-but-still-running", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			staysOn: true, applyFails: true, actions: "[power:off]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "not stopped after the power-off"},
		{name: "power-off-activity-fails", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failPower: "off", applyFails: true, actions: "[power:off]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "may still be in progress"},
		// A confirmed shutdown followed by a failure that wrote nothing DOES give
		// the machine back — the mirror image of the invariant below.
		{name: "drift-during-shutdown-restores-power", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, applyFails: true, actions: "[power:off power:on]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "changed while the virtual machine was being powered off"},

		// --- the two load-bearing invariants ----------------------------------
		// A write whose outcome is unknown must never be booted into.
		{name: "write-activity-failure-stays-stopped", wantState: "off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failWrite: "DELETE", applyFails: true, actions: "[power:off POST DELETE]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true},
		// Same, when the HTTP call itself fails: mutated must already be set
		// BEFORE the request, not after a successful return.
		{name: "write-call-failure-stays-stopped", wantState: "off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failCall: "POST", applyFails: true, actions: "[power:off POST]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true},
		// A VERIFIED transaction whose power settlement fails must KEEP its new
		// intent: rolling it back would make every later apply refuse forever.
		{name: "verified-then-start-fails-keeps-intent", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			failPower: "on", applyFails: true, actions: "[POST DELETE power:on]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "applied and verified"},
		// A completed power-on activity is not evidence that the machine runs.
		{name: "start-not-confirmed-keeps-intent", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			onNoOp: true, applyFails: true, actions: "[POST DELETE power:on]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "could not be confirmed powered on"},
		// Settlement must not act on a state it cannot interpret: "not the target"
		// is not "the opposite of the target", and powering on could race a
		// shutdown that has not settled.
		{name: "settlement-refuses-transitional-state", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			oddLater: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "unsupported power state, so its power state was left untouched"},
		// The activity id is taken verbatim from a Location header the client does
		// not constrain, so it must never be echoed into a diagnostic.
		{name: "activity-id-is-never-echoed", wantState: "off", statePower: "off", cfgPower: "off", livePower: "stopped",
			secretLoc: true, applyFails: true, actions: "[POST]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true, diagnostic: "not reportable"},
		// "This apply wrote nothing" is NOT "the machine is safe to start". If
		// another writer leaves a duplicate id behind, the provider must not be
		// the actor that boots a guest into it — not even to undo its own
		// shutdown, and not even though the re-plan already named the problem.
		{name: "duplicate-after-shutdown-stays-off", wantState: "off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			dupAfter: true, applyFails: true, actions: "[power:off]", wantLive: "stopped", wantIntent: "b2xk",
			diagnostic: "duplicate entries for a managed cloud_init key"},
		// Same rule once the transaction is verified: settlement sees the
		// duplicate in the very snapshot it reads, so it must not power on.
		{name: "duplicate-before-settlement-stays-off", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			dupSettle: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped", wantIntent: "bmV3",
			diagnostic: "duplicate entries for a managed cloud_init key"},
		// Not only duplicates: any divergence from the snapshot this apply proved
		// means another writer is active, and the guest must not boot into it.
		{name: "vapp-edited-before-settlement-stays-off", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			modSettle: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped", wantIntent: "bmV3",
			diagnostic: "no longer matches the state this apply verified"},
		// A transitional state is not a stopped one, on the compensation path too.
		{name: "transitional-before-restore-stays-off", wantState: "off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, oddRestor: true, applyFails: true, actions: "[power:off]", wantLive: "stopped",
			wantIntent: "b2xk", diagnostic: "not observably stopped"},
		// The same discipline on the compensation path: a power-on that does not
		// take effect must be reported as unconfirmed, not as a restored machine.
		{name: "restore-not-confirmed-is-reported", wantState: "off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, onNoOp: true, applyFails: true, actions: "[power:off power:on]", wantLive: "stopped",
			wantIntent: "b2xk", diagnostic: "could not be confirmed running again"},

		// --- M2: an observation nobody made must never be recorded ------------
		// The machine was ALREADY stopped, so no shutdown was needed and the
		// provider never touched its power — yet the write failed, and the SDK
		// persists the planned "on". The live read that found it stopped is an
		// observation and has to be recorded, or the state claims the one thing
		// that is certainly false: that a machine nobody started is running.
		{name: "already-stopped-write-fails-records-off", wantState: "off", statePower: "off", cfgPower: "on", livePower: "stopped",
			failCall: "POST", applyFails: true, actions: "[POST]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true},
		// The mirror image, on the compensation path: the provider powered the
		// machine off and recorded "off", then ANOTHER actor started it while it
		// was down. The safety check sees "running" and refuses to act, so the
		// provider's own "off" is now stale — recording it would describe a
		// machine that is running as stopped, and the next plan would propose to
		// start a machine that is already up.
		{name: "external-start-is-observed-not-overwritten", wantState: "on", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, extStart: true, applyFails: true, actions: "[power:off]", wantLive: "running",
			wantIntent: "b2xk", diagnostic: "not observably stopped"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceVirtualMachine()
			state, cfg := cloudInitStateConfig()
			state.Attributes["power_state"] = tc.statePower
			cfg.Raw["power_state"] = tc.cfgPower
			if tc.allow {
				cfg.Raw["allow_vm_restart"] = true
			}
			switch tc.otherAttr {
			case "name":
				cfg.Raw["name"] = "new-name"
			case "tags":
				cfg.Raw["tags"] = map[string]interface{}{"env": "prod"}
			}
			if tc.unsafe {
				cfg.Raw["cloud_init"] = map[string]interface{}{"user-data": "unsafe/value"}
			}
			cfg.Config = cfg.Raw

			liveValue := tc.liveValue
			if liveValue == "" {
				liveValue = "b2xk"
			}
			f := &cloudInitPowerFake{t: t, power: tc.livePower,
				failPower: tc.failPower, failCall: tc.failCall, failWrite: tc.failWrite,
				failRead: tc.failRead, staysOn: tc.staysOn, onNoOp: tc.onNoOp, drift: tc.drift,
				oddLater: tc.oddLater, secretLoc: tc.secretLoc,
				dupSettle: tc.dupSettle, modSettle: tc.modSettle, dupAfter: tc.dupAfter, oddRestor: tc.oddRestor,
				extStart: tc.extStart,
				props: []map[string]interface{}{{"key": float64(7000), "id": "user-data", "value": liveValue,
					"label": "User data", "type": "string", "userConfigurable": float64(1)}}}

			var logs bytes.Buffer
			ctx := tflogtest.RootLogger(context.Background(), &logs)
			c := newAssignTestClient(t, f.handler)

			diff, err := r.Diff(ctx, state, cfg, nil)
			if tc.planFails {
				if err == nil {
					t.Fatal("a running machine without allow_vm_restart must be refused at plan time")
				}
				if !strings.Contains(err.Error(), tc.diagnostic) {
					t.Fatalf("plan refusal must name its own cause; got: %v", err)
				}
				if len(f.actions) != 0 {
					t.Fatalf("plan-time refusal touched the platform: %v", f.actions)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			next, diags := r.Apply(ctx, state, diff, c)
			if diags.HasError() != tc.applyFails {
				t.Fatalf("apply error = %v, want %v (%v)", diags.HasError(), tc.applyFails, diags)
			}
			if got := fmt.Sprint(f.actions); got != tc.actions {
				t.Fatalf("platform actions = %s, want %s", got, tc.actions)
			}
			if f.power != tc.wantLive {
				t.Fatalf("live power = %q, want %q", f.power, tc.wantLive)
			}
			message := fmt.Sprint(diags)
			if strings.Contains(message+logs.String(), "secret-from-api") {
				t.Fatal("a power or activity error leaked backend content into diagnostics or logs")
			}
			// Asserting "it failed" is not enough when several guards could have
			// produced the failure: pin each one to its own diagnostic, or
			// removing it still passes because a later check catches the case.
			if tc.diagnostic != "" && !strings.Contains(message, tc.diagnostic) {
				t.Fatalf("failure must be attributable to its own guard (want %q); got: %s", tc.diagnostic, message)
			}
			if got := strings.Contains(message, "Do not reapply"); got != tc.recovery {
				t.Fatalf("recovery alarm = %v, want %v: it must fire when, and only when, a vApp write was attempted", got, tc.recovery)
			}
			if next.Attributes["cloud_init.user-data"] != tc.wantIntent {
				t.Fatalf("persisted cloud_init = %q, want %q", next.Attributes["cloud_init.user-data"], tc.wantIntent)
			}
			// A FAILED apply is exactly where the recorded power state matters:
			// the SDK persists PLANNED values on error, so exempting failures
			// from this check is what let a machine deliberately left stopped be
			// recorded as running.
			if next.Attributes["power_state"] != tc.wantState {
				t.Fatalf("recorded power_state = %q, want %q", next.Attributes["power_state"], tc.wantState)
			}
			if tc.applyFails {
				return
			}
			after, err := r.Diff(context.Background(), next, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			if after != nil && (after.Attributes["cloud_init.user-data"] != nil || after.Attributes["power_state"] != nil) {
				t.Fatal("a successful update must converge")
			}
		})
	}
}

// TestVMwareCloudInitVerifiedIntentConverges is the convergence half of the
// rollback rule, and it is the reason that rule exists.
//
// When the vApp transaction is verified and only the power settlement fails,
// the applied intent stays in state. Restoring the old map instead would be
// PERMANENT rather than merely untidy. The refresh cannot undo it: it never
// adds a key back, and never removes one it can no longer observe. So the next
// plan would ask to add a key the vApp already has (which the planner refuses
// to adopt) or to restore one it no longer has (which the planner refuses as
// lost), for ever. Each shape is exercised separately
// because a fix that only kept replacements would pass a replacement-only test.
func TestVMwareCloudInitVerifiedIntentConverges(t *testing.T) {
	for _, tc := range []struct {
		name      string
		stateMap  map[string]string // prior cloud_init in state
		configMap map[string]interface{}
		liveProps []map[string]interface{}
		writes    string
	}{
		{
			name:      "add-a-new-key",
			stateMap:  map[string]string{"cloud_init.%": "1", "cloud_init.user-data": "b2xk"},
			configMap: map[string]interface{}{"user-data": "b2xk", "hostname": "web01"},
			liveProps: []map[string]interface{}{{"key": float64(7000), "id": "user-data", "value": "b2xk",
				"label": "User data", "type": "string", "userConfigurable": float64(1)}},
			writes: "[POST power:on]",
		},
		{
			name: "remove-a-managed-key",
			stateMap: map[string]string{"cloud_init.%": "2", "cloud_init.user-data": "b2xk",
				"cloud_init.hostname": "web01"},
			configMap: map[string]interface{}{"user-data": "b2xk"},
			liveProps: []map[string]interface{}{
				{"key": float64(7000), "id": "user-data", "value": "b2xk", "label": "User data", "type": "string", "userConfigurable": float64(1)},
				{"key": float64(7005), "id": "hostname", "value": "web01", "label": "Hostname", "type": "string", "userConfigurable": float64(1)},
			},
			writes: "[DELETE power:on]",
		},
		{
			name:      "replace-a-value",
			stateMap:  map[string]string{"cloud_init.%": "1", "cloud_init.user-data": "b2xk"},
			configMap: map[string]interface{}{"user-data": "bmV3"},
			liveProps: []map[string]interface{}{{"key": float64(7000), "id": "user-data", "value": "b2xk",
				"label": "User data", "type": "string", "userConfigurable": float64(1)}},
			writes: "[POST DELETE power:on]",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceVirtualMachine()
			state, cfg := cloudInitStateConfig()
			for k := range state.Attributes {
				if strings.HasPrefix(k, "cloud_init.") {
					delete(state.Attributes, k)
				}
			}
			for k, v := range tc.stateMap {
				state.Attributes[k] = v
			}
			state.Attributes["power_state"] = "off"
			cfg.Raw["cloud_init"] = tc.configMap
			cfg.Raw["power_state"] = "on"
			cfg.Config = cfg.Raw

			// The vApp transaction succeeds; only the final power-on fails.
			f := &cloudInitPowerFake{t: t, power: "stopped", failPower: "on", props: tc.liveProps}
			c := newAssignTestClient(t, f.handler)

			diff, err := r.Diff(context.Background(), state, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			next, diags := r.Apply(context.Background(), state, diff, c)
			if !diags.HasError() {
				t.Fatal("a failed power-on must fail the apply")
			}
			if got := fmt.Sprint(f.actions); got != tc.writes {
				t.Fatalf("platform actions = %s, want %s", got, tc.writes)
			}
			if msg := fmt.Sprint(diags); !strings.Contains(msg, "applied and verified") {
				t.Fatalf("a verified transaction must say so rather than raise the recovery alarm; got: %s", msg)
			}
			if strings.Contains(fmt.Sprint(diags), "Do not reapply") {
				t.Fatal("a verified transaction must not raise the uncertain-mutation recovery alarm")
			}
			// The applied intent is kept, so the next plan has nothing left to do
			// on cloud_init — only the power state remains to converge. Had the
			// intent been rolled back, the planner would refuse this for ever.
			after, err := r.Diff(context.Background(), next, cfg, nil)
			if err != nil {
				t.Fatalf("the retry plan must not be refused: %v", err)
			}
			for k := range tc.configMap {
				if after != nil && after.Attributes["cloud_init."+k] != nil {
					t.Fatalf("cloud_init.%s still diverges: the verified transaction was rolled back", k)
				}
			}
			if after == nil || after.Attributes["power_state"] == nil {
				t.Fatal("the unconverged power state must still be planned")
			}
		})
	}
}

// cloudInitFakeChanges answers the sole-change rule without a diff engine.
type cloudInitFakeChanges map[string]bool

func (f cloudInitFakeChanges) HasChange(name string) bool { return f[name] }

// TestVMwareCloudInitSoleChangeRule pins which attributes may travel with
// cloud_init. The rule is evaluated twice — brought forward to the plan so the
// operator learns before an apply begins, and again at apply where it is
// authoritative because the plan copy fails open on unresolved values — so it
// lives in one function and is pinned here directly rather than only through
// whichever of the two happens to fire first.
func TestVMwareCloudInitSoleChangeRule(t *testing.T) {
	// Derived from the schema rather than hard-coded, so the case keeps meaning
	// if the attribute it picks is ever renamed.
	computedOnly := ""
	for name, sch := range resourceVirtualMachine().Schema {
		if sch.Computed && !sch.Optional && !sch.Required {
			if computedOnly == "" || name < computedOnly {
				computedOnly = name
			}
		}
	}
	if computedOnly == "" {
		t.Fatal("the schema has no computed-only attribute; this case can no longer be expressed")
	}
	for _, tc := range []struct {
		name    string
		changed []string
		want    string
	}{
		{name: "cloud_init alone", changed: []string{"cloud_init"}},
		// The opt-in has to be grantable in the same apply as the change it
		// authorises, and this path drives power itself.
		{name: "power_state may travel with it", changed: []string{"cloud_init", "power_state"}},
		{name: "allow_vm_restart may travel with it", changed: []string{"cloud_init", "allow_vm_restart"}},
		{name: "both power attributes may travel with it", changed: []string{"cloud_init", "power_state", "allow_vm_restart"}},
		{name: "any other attribute is named", changed: []string{"cloud_init", "name"}, want: "name"},
		{name: "a computed-only attribute is not a change the operator made", changed: []string{"cloud_init", computedOnly}},
		// Schema iteration order is random. Without sorting, the diagnostic named
		// a different attribute from run to run, which no operator can script
		// against and no test can pin.
		{name: "several changed attributes always resolve to the same one", changed: []string{"cloud_init", "tags", "name", "memory"}, want: "memory"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			set := cloudInitFakeChanges{}
			for _, n := range tc.changed {
				set[n] = true
			}
			if got := cloudInitOtherChangedAttribute(set); got != tc.want {
				t.Fatalf("cloudInitOtherChangedAttribute = %q, want %q", got, tc.want)
			}
		})
	}
}

// TestVMwareCloudInitPlanGuardRawConfig exercises the plan guard against a REAL
// configuration object rather than the legacy shim.
//
// TestVMwareCloudInitPlanGuardUnknowns builds its configuration with
// terraform.NewResourceConfigRaw, which leaves GetRawConfig null — so the
// raw-configuration branch of the guard is skipped entirely there, and whatever
// it does or does not catch, that table cannot tell. These cases go through
// NewResourceConfigShimmed, where the guard sees the cty object Terraform
// actually passes.
//
// The configuration is built from the schema's own implied type rather than
// written out attribute by attribute, so it keeps working when the schema gains
// one.
func TestVMwareCloudInitPlanGuardRawConfig(t *testing.T) {
	block := resourceVirtualMachine().CoreConfigSchema()
	config := func(cloudInit cty.Value, extra map[string]cty.Value) *terraform.ResourceConfig {
		vals := map[string]cty.Value{}
		for name, ty := range block.ImpliedType().AttributeTypes() {
			vals[name] = cty.NullVal(ty)
		}
		vals["name"] = cty.StringVal("investigation")
		vals["datacenter_id"] = cty.StringVal("dc")
		vals["host_cluster_id"] = cty.StringVal("hc")
		vals["guest_operating_system_moref"] = cty.StringVal("ubuntu64Guest")
		vals["power_state"] = cty.StringVal("on")
		vals["cloud_init"] = cloudInit
		for k, v := range extra {
			vals[k] = v
		}
		return terraform.NewResourceConfigShimmed(cty.ObjectVal(vals), block)
	}
	known := cty.MapVal(map[string]cty.Value{"user-data": cty.StringVal("bmV3")})

	for _, tc := range []struct {
		name      string
		cloudInit cty.Value
		extra     map[string]cty.Value
		refuse    bool
	}{
		// The one refusal: everything resolved, machine running, no opt-in.
		{name: "running, changed, no opt-in", cloudInit: known, refuse: true},
		{name: "opt-in granted", cloudInit: known,
			extra: map[string]cty.Value{"allow_vm_restart": cty.True}},
		// The cases the raw-configuration branch exists for. An unknown ELEMENT
		// leaves the map's element count known, so an attribute-level check has
		// nothing to notice; only the configuration object shows it.
		{name: "an unknown element inside the map", cloudInit: cty.MapVal(map[string]cty.Value{
			"user-data": cty.UnknownVal(cty.String)})},
		{name: "one known and one unknown element", cloudInit: cty.MapVal(map[string]cty.Value{
			"user-data": cty.StringVal("bmV3"), "hostname": cty.UnknownVal(cty.String)})},
		{name: "the whole map unknown", cloudInit: cty.UnknownVal(cty.Map(cty.String))},
		{name: "an unknown power_state", cloudInit: known,
			extra: map[string]cty.Value{"power_state": cty.UnknownVal(cty.String)}},
		{name: "an unknown opt-in", cloudInit: known,
			extra: map[string]cty.Value{"allow_vm_restart": cty.UnknownVal(cty.Bool)}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Everything the configuration sets must already be in state, or the
			// sole-change rule fires and masks what this table is about.
			state := &terraform.InstanceState{ID: "vm-1", Attributes: map[string]string{
				"id": "vm-1", "name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
				"guest_operating_system_moref": "ubuntu64Guest",
				"power_state":                  "on", "cloud_init.%": "1", "cloud_init.user-data": "b2xk",
			}}
			_, err := resourceVirtualMachine().Diff(context.Background(), state, config(tc.cloudInit, tc.extra), nil)
			if tc.refuse {
				if err == nil || !strings.Contains(err.Error(), "allow_vm_restart") {
					t.Fatalf("expected the power refusal, got: %v", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("the guard must fail open here, got: %v", err)
			}
		})
	}
}

// TestVerifyVAppAdditionContract pins what the provider will accept as proof
// that its own addition took effect.
//
// The end-to-end tests cannot do this. Their fake servers append the submitted
// descriptor to the snapshot unchanged, so the "after" they return is derived
// from the payload and verifyVAppAddition can never disagree with it — the
// check passes by construction rather than by being right. The real API is
// under no such obligation: it assigns the numeric key itself, and nothing in
// this repository establishes that it echoes every other field verbatim.
//
// So the contract is pinned here against hand-built snapshots, where "after"
// owes nothing to the payload.
func TestVerifyVAppAdditionContract(t *testing.T) {
	existing := func() map[string]interface{} {
		return map[string]interface{}{"key": float64(1), "id": "hostname", "value": "before",
			"label": "Hostname", "type": "string", "userConfigurable": float64(1)}
	}
	added := func() map[string]interface{} {
		return map[string]interface{}{"key": float64(2), "id": "user-data", "value": "bmV3",
			"label": "User data", "type": "string", "userConfigurable": float64(1)}
	}
	payload := map[string]interface{}{"id": "user-data", "value": "bmV3",
		"label": "User data", "type": "string", "userConfigurable": true}

	for _, tc := range []struct {
		name   string
		mutate func(after *client.VAppSnapshot)
		accept bool
		// Which refusal must fire. Asserting only "it errored" lets a different
		// check pass for the wrong reason, after which removing the one under
		// test still leaves the suite green.
		because string
	}{
		{name: "a faithful addition", accept: true},
		// Identity, power and vApp state are part of the proof: a snapshot of
		// another machine, or of this one no longer stopped, proves nothing
		// about what was just written.
		{name: "the snapshot belongs to another machine", because: "unexpected addition result", mutate: func(a *client.VAppSnapshot) { a.ID = "vm-other" }},
		{name: "the machine is no longer stopped", because: "unexpected addition result", mutate: func(a *client.VAppSnapshot) { a.PowerState = "running" }},
		{name: "the vApp is no longer enabled", because: "unexpected addition result", mutate: func(a *client.VAppSnapshot) { a.Enabled = false }},
		{name: "nothing was added", because: "unexpected addition result", mutate: func(a *client.VAppSnapshot) {
			a.Properties = a.Properties[:1]
		}},
		{name: "more than one property appeared", because: "unexpected addition result", mutate: func(a *client.VAppSnapshot) {
			extra := added()
			extra["key"] = float64(3)
			extra["id"] = "seedfrom"
			a.Properties = append(a.Properties, extra)
		}},
		// An unrelated property that moved means another writer is active, and
		// the sequence can no longer reason about what it is deleting next.
		{name: "an existing descriptor changed", because: "existing descriptor changed", mutate: func(a *client.VAppSnapshot) {
			a.Properties[0]["value"] = "changed-by-someone-else"
		}},
		// The API assigns the numeric key, so that one field is expected to
		// differ. Everything else must come back as it was sent.
		{name: "the added descriptor has a different id", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			a.Properties[1]["id"] = "user_data"
		}},
		{name: "the added descriptor lost its label", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			delete(a.Properties[1], "label")
		}},
		{name: "the added descriptor has a different value", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			a.Properties[1]["value"] = "something-else"
		}},
		{name: "the added descriptor flipped userConfigurable", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			a.Properties[1]["userConfigurable"] = float64(0)
		}},
		{name: "the added descriptor carries a field the API cannot replay", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			a.Properties[1]["typeReference"] = "something"
		}},
		// The count check is the only thing standing between this and silent
		// data loss: the loop would find the added descriptor, set found, and
		// return success while an existing property had vanished.
		{name: "an existing property vanished as the new one appeared", because: "unexpected addition result",
			mutate: func(a *client.VAppSnapshot) {
				a.Properties = a.Properties[1:]
			}},
		// Two descriptors matching the payload is not success twice over: it is
		// the duplicate state the delete step is about to reason against.
		{name: "the descriptor was added twice", because: "added descriptor differs", mutate: func(a *client.VAppSnapshot) {
			twin := added()
			twin["key"] = float64(3)
			a.Properties = append(a.Properties, twin)
			a.Properties[0] = twin // keep the count at before+1 by displacing the existing one
			a.Properties = a.Properties[1:]
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := &client.VAppSnapshot{ID: "vm-1", PowerState: "stopped", Enabled: true,
				Properties: []map[string]interface{}{existing()}}
			after := &client.VAppSnapshot{ID: "vm-1", PowerState: "stopped", Enabled: true,
				Properties: []map[string]interface{}{existing(), added()}}
			if tc.mutate != nil {
				tc.mutate(after)
			}
			err := verifyVAppAddition(before, after, payload)
			if tc.accept && err != nil {
				t.Fatalf("a faithful addition must be accepted; got: %v", err)
			}
			if !tc.accept {
				if err == nil {
					t.Fatal("accepted an addition the provider cannot prove it made")
				}
				if !strings.Contains(err.Error(), tc.because) {
					t.Fatalf("refused for the wrong reason: want %q, got %q", tc.because, err)
				}
			}
		})
	}
}

// TestVMwareCloudInitPlanGuardUnknowns pins the plan-time guard's edges. It must
// refuse exactly one situation and fail OPEN on everything it cannot decide:
// a value Terraform has not resolved yet must never turn into a refusal.
func TestVMwareCloudInitPlanGuardUnknowns(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	// Built as two named values rather than one two-literal return: gofmt 1.24
	// and later releases disagree on how to indent the latter, and CI formats
	// with the toolchain named in go.mod.
	base := func() (*terraform.InstanceState, map[string]interface{}) {
		state := &terraform.InstanceState{ID: "vm-1", Attributes: map[string]string{
			"id": "vm-1", "name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "on", "cloud_init.%": "1", "cloud_init.user-data": "b2xk",
		}}
		cfg := map[string]interface{}{
			"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "on", "cloud_init": map[string]interface{}{"user-data": "bmV3"},
		}
		return state, cfg
	}
	for _, tc := range []struct {
		name       string
		mutate     func(*terraform.InstanceState, map[string]interface{})
		refuse     bool
		refuseWith string // the substring the refusal must carry; defaults to the power one
	}{
		{name: "running, changed, no opt-in => the one refusal", refuse: true},
		{name: "create", mutate: func(st *terraform.InstanceState, _ map[string]interface{}) { st.ID = "" }},
		{name: "cloud_init unchanged", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["cloud_init"] = map[string]interface{}{"user-data": "b2xk"}
		}},
		{name: "machine already stopped", mutate: func(st *terraform.InstanceState, _ map[string]interface{}) {
			st.Attributes["power_state"] = "off"
		}},
		{name: "being powered off anyway", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["power_state"] = "off"
		}},
		{name: "opt-in granted", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["allow_vm_restart"] = true
		}},
		{name: "unknown cloud_init map", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["cloud_init"] = unknown
		}},
		{name: "unknown element inside the map", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["cloud_init"] = map[string]interface{}{"user-data": unknown}
		}},
		{name: "unknown power_state", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["power_state"] = unknown
		}},
		{name: "unknown opt-in", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["allow_vm_restart"] = unknown
		}},
		// A ForceNew change asks for a REPLACEMENT, not an in-place update, so
		// the in-place restart policy does not apply to it. CustomizeDiff runs
		// with the existing id before the SDK resolves the replacement, so the
		// guard has to recognise that itself or it refuses a legitimate plan.
		{name: "replacement via a ForceNew change", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["deploy_options"] = map[string]interface{}{"disk_provisioning_type": "thin"}
		}},
		// The sole-change rule, now reported at plan time instead of after the
		// update path has been entered.
		{name: "another attribute changed in the same apply", refuse: true, refuseWith: `"name"`,
			mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
				c["allow_vm_restart"] = true // so the power guard cannot be what refuses
				c["name"] = "renamed"
			}},
		// ...but only once its new value is resolved. An unknown reports as
		// changed whether or not it will be, and the apply-time copy decides.
		{name: "another attribute unknown in the same apply", mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
			c["allow_vm_restart"] = true
			c["name"] = unknown
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, raw := base()
			if tc.mutate != nil {
				tc.mutate(st, raw)
			}
			_, err := resourceVirtualMachine().Diff(context.Background(), st, terraform.NewResourceConfigRaw(raw), nil)
			switch {
			case tc.refuse && func() bool {
				want := tc.refuseWith
				if want == "" {
					want = "allow_vm_restart"
				}
				return err == nil || !strings.Contains(err.Error(), want)
			}():
				t.Fatalf("expected the cloud_init refusal to name its own cause, got: %v", err)
			case !tc.refuse && err != nil:
				t.Fatalf("the guard must fail open here, got: %v", err)
			}
		})
	}
}

// TestVMwareCloudInitDeployOptionsCollision pins the plan-time refusal of a key
// declared in both maps.
//
// Creation merges cloud_init into deploy_options and gives deploy_options
// precedence, so a shared key's cloud_init value is dropped before the deploy
// request is built and never reaches the machine — while state records it as
// applied. The update path refuses a shared key as well, but only once the
// apply has begun and only for a key that CHANGES. Neither reports an
// unchanging collision, and neither reports anything at all on create.
//
// The refusal is scoped to the OVERLAP, not to using both attributes: cloud_init
// alongside unrelated deploy_options is the ordinary case, and deploy_options
// may own a cloud-init key outright as long as cloud_init does not claim it too.
func TestVMwareCloudInitDeployOptionsCollision(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	base := func() (*terraform.InstanceState, map[string]interface{}) {
		state := &terraform.InstanceState{ID: "vm-1", Attributes: map[string]string{
			"id": "vm-1", "name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "off", "cloud_init.%": "1", "cloud_init.user-data": "b2xk",
		}}
		cfg := map[string]interface{}{
			"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "off", "cloud_init": map[string]interface{}{"user-data": "bmV3"},
		}
		return state, cfg
	}
	for _, tc := range []struct {
		name    string
		mutate  func(*terraform.InstanceState, map[string]interface{})
		refuse  []string // keys the collision diagnostic must name, in order
		restart bool     // expects the power-state refusal instead
	}{
		{
			name: "a shared key is refused on update",
			mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
				c["deploy_options"] = map[string]interface{}{"user-data": "owned-by-deploy"}
			},
			refuse: []string{"user-data"},
		},
		{
			// The case nothing covered before: creation is where the value is
			// actually dropped, and it had no check at all.
			name: "a shared key is refused on create",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				st.ID = ""
				c["deploy_options"] = map[string]interface{}{"user-data": "owned-by-deploy"}
			},
			refuse: []string{"user-data"},
		},
		{
			// The apply-time refusal only looks at CHANGED keys, so a collision
			// already sitting in state was silent for the whole life of the
			// resource. Nothing about cloud_init or deploy_options moves here —
			// only an unrelated attribute does, which is what an operator
			// actually does to a machine deployed years ago.
			//
			// deploy_options must be identical in state AND config: it is
			// ForceNew, and declaring it fresh would make this a REPLACEMENT,
			// where the SDK reports every attribute as changing and the case
			// would no longer isolate anything.
			name: "a collision already in state is refused when nothing about it changes",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				st.Attributes["deploy_options.%"] = "1"
				st.Attributes["deploy_options.user-data"] = "legacy"
				c["deploy_options"] = map[string]interface{}{"user-data": "legacy"}
				c["cloud_init"] = map[string]interface{}{"user-data": "b2xk"}
				c["name"] = "renamed" // the only real change in this plan
			},
			refuse: []string{"user-data"},
		},
		{
			// Determinism: map iteration order must not reorder the message.
			name: "every shared key is named, in a stable order",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				st.Attributes["cloud_init.%"] = "2"
				st.Attributes["cloud_init.hostname"] = "web01"
				c["cloud_init"] = map[string]interface{}{"user-data": "bmV3", "hostname": "web02"}
				c["deploy_options"] = map[string]interface{}{
					"user-data": "a", "hostname": "b", "disk_provisioning_type": "thin"}
			},
			refuse: []string{"hostname", "user-data"},
		},
		{
			// The ordinary case the operator asked about: both attributes used,
			// no overlap. This must stay valid.
			name: "unrelated deploy_options alongside cloud_init pass",
			mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
				c["deploy_options"] = map[string]interface{}{"disk_provisioning_type": "thin"}
			},
		},
		{
			// deploy_options may own a cloud-init key outright. Only cloud_init
			// ALSO claiming it creates the ambiguity.
			name: "a cloud-init key owned only by deploy_options passes",
			mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
				c["deploy_options"] = map[string]interface{}{"hostname": "web01"}
			},
		},
		{
			// Dropping the key from cloud_init is the only non-destructive way
			// out of a collision, and it plans no vApp write: the key is
			// skipped, not deleted. Demanding the restart opt-in for it would
			// make the remedy cost an interruption to change nothing — on a
			// running machine, which is the case that needs the remedy most.
			name: "resolving a collision on a running machine needs no opt-in",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				st.Attributes["power_state"] = "on"
				st.Attributes["deploy_options.%"] = "1"
				st.Attributes["deploy_options.user-data"] = "legacy"
				c["power_state"] = "on"
				c["deploy_options"] = map[string]interface{}{"user-data": "legacy"}
				delete(c, "cloud_init")
			},
		},
		{
			// The exemption is confined to deploy-owned keys. An ordinary
			// removal does write to the vApp, so it still needs the machine
			// stopped — the exemption must not become a general bypass.
			name: "removing an ordinary key on a running machine still needs the opt-in",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				st.Attributes["power_state"] = "on"
				st.Attributes["deploy_options.%"] = "1"
				st.Attributes["deploy_options.disk_provisioning_type"] = "thin"
				c["power_state"] = "on"
				c["deploy_options"] = map[string]interface{}{"disk_provisioning_type": "thin"}
				delete(c, "cloud_init")
			},
			restart: true,
		},
		{
			name: "cloud_init alone passes",
		},
		{
			name: "deploy_options alone passes",
			mutate: func(st *terraform.InstanceState, c map[string]interface{}) {
				delete(st.Attributes, "cloud_init.%")
				delete(st.Attributes, "cloud_init.user-data")
				delete(c, "cloud_init")
				c["deploy_options"] = map[string]interface{}{"user-data": "owned-by-deploy"}
			},
		},
		{
			// An unresolved map reads as empty here, so the check skips rather
			// than refusing a plan Terraform has not finished resolving.
			name: "an unknown cloud_init map does not refuse",
			mutate: func(_ *terraform.InstanceState, c map[string]interface{}) {
				c["cloud_init"] = unknown
				c["deploy_options"] = map[string]interface{}{"user-data": "owned-by-deploy"}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, raw := base()
			if tc.mutate != nil {
				tc.mutate(st, raw)
			}
			_, err := resourceVirtualMachine().Diff(context.Background(), st, terraform.NewResourceConfigRaw(raw), nil)
			if tc.restart {
				if err == nil || !strings.Contains(err.Error(), "allow_vm_restart") {
					t.Fatalf("expected the power-state refusal, got: %v", err)
				}
				return
			}
			if len(tc.refuse) == 0 {
				if err != nil {
					t.Fatalf("the guard must not refuse here, got: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected the collision refusal, got none")
			}
			msg := err.Error()
			if !strings.Contains(msg, "both cloud_init and deploy_options") {
				t.Fatalf("the refusal must name the two attributes; got: %s", msg)
			}
			// Pinned as one substring so the order is part of the contract.
			if want := strings.Join(tc.refuse, ", "); !strings.Contains(msg, want) {
				t.Fatalf("the refusal must name %q, in that order; got: %s", want, msg)
			}
		})
	}
}

// TestVMwareCloudInitRefreshResyncsTrackedKeys pins the refresh contract.
//
// Before this, `cloud_init` was a write-only record of the last applied intent:
// an out-of-band change to a managed key was invisible for ever, which is not
// what Terraform state means. It now resynchronises, within four bounds that
// each exist because crossing it breaks an ordinary, unchanged configuration:
//
//   - it never adopts an untracked property — a machine's vApp also carries
//     values from its image, from deploy_options and from the platform;
//   - it never drops a tracked key. Clone and from-scratch creations send no
//     deploy options, so those machines legitimately track keys their vApp never
//     received; dropping them makes an unchanged configuration plan, and that
//     plan is refused on a running machine without allow_vm_restart;
//   - it never touches a key deploy_options also declares, because creation
//     sends the deploy_options value and the planner refuses shared keys;
//   - it never fails, per key or at all. A failed refresh takes the whole plan
//     with it and offers no provider-side recovery.
func TestVMwareCloudInitRefreshResyncsTrackedKeys(t *testing.T) {
	prop := func(key float64, id, value string) map[string]interface{} {
		return map[string]interface{}{"key": key, "id": id, "value": value,
			"label": id, "type": "string", "userConfigurable": float64(1)}
	}
	for _, tc := range []struct {
		name      string
		tracked   map[string]string
		deploy    map[string]string
		props     []map[string]interface{}
		failGetNo int
		// The two response shapes the general read tolerates and the vApp
		// snapshot does not: HTTP 206, and a null property list.
		code206No   int
		nullPropsNo int
		wantWarn    string
		want        map[string]string // expected cloud_init in state afterwards
	}{
		{
			name:    "a value changed out of band becomes visible",
			tracked: map[string]string{"user-data": "b2xk"},
			props:   []map[string]interface{}{prop(7000, "user-data", "changed-outside")},
			want:    map[string]string{"user-data": "changed-outside"},
		},
		{
			// Measured on the platform: creation accepts a value the update
			// path refuses, and stores a normalised form — '/' comes back
			// HTML-escaped. Recording that form would raise a diff against the
			// declared value which the planner then refuses to write, for ever,
			// and on a running machine the plan itself would fail. The recorded
			// value stays, which is what 1.12 did and it converged.
			name:    "a recorded value carrying '/' is never resynchronised",
			tracked: map[string]string{"user-data": "YWJj/ZGVm"},
			props:   []map[string]interface{}{prop(7000, "user-data", "YWJj&#x2F;ZGVm")},
			want:    map[string]string{"user-data": "YWJj/ZGVm"},
		},
		{
			// Same rule, the other normalisation the platform was seen to apply.
			name:    "a recorded value with surrounding whitespace is never resynchronised",
			tracked: map[string]string{"hostname": " web01 "},
			props:   []map[string]interface{}{prop(7000, "hostname", "web01")},
			want:    map[string]string{"hostname": " web01 "},
		},
		{
			// The bound is per key, and it must not spill: an unwritable key
			// freezes only itself.
			name:    "an unwritable key does not freeze a writable one",
			tracked: map[string]string{"user-data": "YWJj/ZGVm", "hostname": "web01"},
			props: []map[string]interface{}{prop(7000, "user-data", "YWJj&#x2F;ZGVm"),
				prop(7001, "hostname", "renamed-outside")},
			want: map[string]string{"user-data": "YWJj/ZGVm", "hostname": "renamed-outside"},
		},
		{
			// The whole point of the boundary: the machine carries a property
			// the operator never declared, and refresh must not claim it.
			name:    "untracked live properties are never adopted",
			tracked: map[string]string{"user-data": "b2xk"},
			props: []map[string]interface{}{prop(7000, "user-data", "b2xk"),
				prop(7001, "hostname", "set-by-the-image"), prop(7002, "seedfrom", "http-ish")},
			want: map[string]string{"user-data": "b2xk"},
		},
		{
			// A clone or from-scratch machine tracks keys its vApp never got,
			// because neither creation path sends deploy options. Dropping them
			// would make this unchanged configuration plan.
			name:    "a key absent from the vApp is kept, not dropped",
			tracked: map[string]string{"user-data": "b2xk", "hostname": "web01"},
			props:   []map[string]interface{}{prop(7000, "user-data", "b2xk")},
			want:    map[string]string{"user-data": "b2xk", "hostname": "web01"},
		},
		{
			// Creation sends the deploy_options value for a shared key, so the
			// live property belongs to that attribute. Recording it here leaves
			// a diff the planner then refuses for collision, with no way out.
			name:    "a key shared with deploy_options is never resynchronised",
			tracked: map[string]string{"user-data": "declared-in-cloud-init"},
			deploy:  map[string]string{"user-data": "declared-in-deploy-options"},
			props:   []map[string]interface{}{prop(7000, "user-data", "declared-in-deploy-options")},
			want:    map[string]string{"user-data": "declared-in-cloud-init"},
		},
		{
			// The general read accepts HTTP 206; the vApp snapshot requires 200.
			// Whether the platform ever answers 206 for a single machine is not
			// established, so what matters is that it degrades rather than
			// taking the plan down with it.
			name:      "a 206 response leaves every recorded value untouched",
			tracked:   map[string]string{"user-data": "b2xk", "hostname": "web01"},
			props:     []map[string]interface{}{prop(7000, "user-data", "changed-outside")},
			code206No: 2,
			wantWarn:  "could not be read",
			want:      map[string]string{"user-data": "b2xk", "hostname": "web01"},
		},
		{
			// Likewise a machine whose vApp is absent: the snapshot refuses an
			// incomplete answer rather than read it as "every property is gone".
			name:        "a null property list leaves every recorded value untouched",
			tracked:     map[string]string{"user-data": "b2xk", "hostname": "web01"},
			props:       []map[string]interface{}{prop(7000, "user-data", "changed-outside")},
			nullPropsNo: 2,
			wantWarn:    "could not be read",
			want:        map[string]string{"user-data": "b2xk", "hostname": "web01"},
		},
		{
			name:      "an unreadable vApp leaves every recorded value untouched",
			tracked:   map[string]string{"user-data": "b2xk", "hostname": "web01"},
			props:     []map[string]interface{}{prop(7000, "user-data", "changed-outside")},
			failGetNo: 2, // the general read succeeds; the vApp re-read does not
			wantWarn:  "could not be read",
			want:      map[string]string{"user-data": "b2xk", "hostname": "web01"},
		},
		{
			name:    "duplicate entries leave that key untouched",
			tracked: map[string]string{"user-data": "b2xk"},
			props: []map[string]interface{}{prop(7000, "user-data", "one"),
				prop(7001, "user-data", "two")},
			wantWarn: "duplicate entries for the cloud_init key",
			want:     map[string]string{"user-data": "b2xk"},
		},
		{
			name:    "a non-string live value leaves that key untouched",
			tracked: map[string]string{"user-data": "b2xk"},
			props: []map[string]interface{}{{"key": float64(7000), "id": "user-data",
				"value": float64(42), "label": "User data", "type": "string"}},
			wantWarn: "does not hold a string value",
			want:     map[string]string{"user-data": "b2xk"},
		},
		{
			// Degradation is per key, not a global bail-out: one unreadable key
			// must not freeze the resynchronisation of a healthy one.
			name:    "one anomalous key does not block another",
			tracked: map[string]string{"user-data": "b2xk", "hostname": "web01"},
			props: []map[string]interface{}{prop(7000, "user-data", "one"),
				prop(7001, "user-data", "two"), prop(7002, "hostname", "renamed-outside")},
			wantWarn: "duplicate entries for the cloud_init key",
			want:     map[string]string{"user-data": "b2xk", "hostname": "renamed-outside"},
		},
		{
			// An imported machine tracks nothing, so there is nothing to
			// resynchronise and, deliberately, nothing to adopt either.
			name:    "nothing tracked adopts nothing",
			tracked: nil,
			props:   []map[string]interface{}{prop(7000, "user-data", "set-by-the-image")},
			want:    nil,
		},
		{
			// A machine that does not use cloud_init at all must not have its
			// refresh made fragile by this feature: the vApp is never read.
			name:      "a machine without cloud_init never reads the vApp",
			tracked:   nil,
			props:     []map[string]interface{}{prop(7000, "user-data", "set-by-the-image")},
			failGetNo: 2,
			want:      nil,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := resourceVirtualMachine()
			state := &terraform.InstanceState{ID: "vm-investigation", Attributes: map[string]string{
				"id": "vm-investigation", "name": "investigation", "datacenter_id": "dc",
				"host_cluster_id": "hc", "guest_operating_system_moref": "ubuntu64Guest",
				"memory": "2147483648", "cpu": "2", "num_cores_per_socket": "1", "power_state": "off",
			}}
			if len(tc.tracked) > 0 {
				state.Attributes["cloud_init.%"] = fmt.Sprint(len(tc.tracked))
				for k, v := range tc.tracked {
					state.Attributes["cloud_init."+k] = v
				}
			}
			if len(tc.deploy) > 0 {
				state.Attributes["deploy_options.%"] = fmt.Sprint(len(tc.deploy))
				for k, v := range tc.deploy {
					state.Attributes["deploy_options."+k] = v
				}
			}
			f := &cloudInitPowerFake{t: t, power: "stopped", props: tc.props, failGetNo: tc.failGetNo,
				code206No: tc.code206No, nullPropsNo: tc.nullPropsNo}
			c := newAssignTestClient(t, f.handler)

			next, diags := r.RefreshWithoutUpgrade(context.Background(), state, c)

			// The invariant that makes every bound above safe to rely on: a
			// refresh anomaly never reaches the operator as a failed plan.
			if diags.HasError() {
				t.Fatalf("refresh must never fail a plan; got: %s", fmt.Sprint(diags))
			}
			warned := ""
			for _, d := range diags {
				if d.Severity != diag.Warning {
					t.Fatalf("unexpected diagnostic severity %v: %s", d.Severity, d.Detail)
				}
				warned += d.Detail + "\n"
			}
			if tc.wantWarn == "" {
				if warned != "" {
					t.Fatalf("expected no warning, got: %s", warned)
				}
			} else if !strings.Contains(warned, tc.wantWarn) {
				t.Fatalf("warning must name its own cause (want %q); got: %q", tc.wantWarn, warned)
			}

			for k, v := range tc.want {
				if got := next.Attributes["cloud_init."+k]; got != v {
					t.Fatalf("cloud_init.%s = %q, want %q", k, got, v)
				}
			}
			for k := range next.Attributes {
				if !strings.HasPrefix(k, "cloud_init.") || k == "cloud_init.%" {
					continue
				}
				if _, declared := tc.want[strings.TrimPrefix(k, "cloud_init.")]; !declared {
					t.Fatalf("refresh adopted %s, which Terraform does not manage", k)
				}
			}
			if got, want := next.Attributes["cloud_init.%"], fmt.Sprint(len(tc.want)); len(tc.want) > 0 && got != want {
				t.Fatalf("cloud_init element count = %q, want %q", got, want)
			}
		})
	}
}

// TestVMwareCloudInitValueValidation pins the creation-time value rule.
//
// Measured on the platform, in three places: the deployment API stores a value
// it has mutated. vCenter held &#x2F; where a '/' was sent; the OVF environment
// handed to the guest carried the escaped form; and an SSH key cloud-init
// installed from it had both of its '/' replaced, so it could never
// authenticate. The control machine, same image and same configuration but a
// key without '/', came out intact.
//
// None of that is reported anywhere: the apply succeeds and the value simply
// never takes effect. The update path already refused these values; this test
// pins the same rule at creation, which is the only place the failure could
// still be introduced.
func TestVMwareCloudInitValueValidation(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	for _, tc := range []struct {
		name   string
		ci     map[string]interface{}
		refuse []string // substrings every diagnostic text must carry, in order
	}{
		{
			name: "an ordinary Base64 payload passes",
			ci:   map[string]interface{}{"user-data": "b2xkCg", "hostname": "web01"},
		},
		{
			// The headline case: this is what reaches the machine as &#x2F;.
			name:   "a value carrying a slash is refused, naming the character",
			ci:     map[string]interface{}{"user-data": "YWJj/ZGVm"},
			refuse: []string{`"user-data"`, `contains '/' at offset 4`},
		},
		{
			// A URL cannot not contain '/', so seedfrom is unusable until the
			// platform stops escaping. Saying so at plan time beats a machine
			// that boots and ignores it.
			name:   "a seedfrom URL is refused",
			ci:     map[string]interface{}{"seedfrom": "http://seed.example.com/data/"},
			refuse: []string{`"seedfrom"`, `contains '/'`},
		},
		{
			name:   "surrounding whitespace is refused",
			ci:     map[string]interface{}{"hostname": " web01 "},
			refuse: []string{`"hostname"`, "leading or trailing whitespace"},
		},
		{
			name:   "an empty value is refused",
			ci:     map[string]interface{}{"hostname": ""},
			refuse: []string{`"hostname"`, "is empty"},
		},
		{
			name:   "an ampersand is refused",
			ci:     map[string]interface{}{"password": "a&b"},
			refuse: []string{`"password"`, `contains '&'`},
		},
		{
			// Map iteration order is random; two bad values must always be
			// reported in the same order or the diagnostics churn between runs.
			name: "several bad values are reported in a stable order",
			ci:   map[string]interface{}{"user-data": "a/b", "hostname": " x "},
			refuse: []string{`"hostname"`, "leading or trailing whitespace",
				`"user-data"`, `contains '/'`},
		},
		{
			// The pre-existing key rule must survive being composed with the
			// value rule.
			name:   "an unknown key is still refused",
			ci:     map[string]interface{}{"not-a-cloud-init-key": "fine"},
			refuse: []string{"not allowed for cloud-init"},
		},
		{
			// A value Terraform has not resolved yet must never be refused: the
			// sentinel carries no forbidden character, so it passes on its own.
			name: "an unresolved value is not refused",
			ci:   map[string]interface{}{"user-data": unknown},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := map[string]interface{}{
				// Validate() enforces the UUID format on these, unlike Diff().
				"name":                         "investigation",
				"datacenter_id":                "11111111-1111-1111-1111-111111111111",
				"host_cluster_id":              "22222222-2222-2222-2222-222222222222",
				"guest_operating_system_moref": "ubuntu64Guest", "memory": 2147483648, "cpu": 2,
				"cloud_init": tc.ci,
			}
			diags := resourceVirtualMachine().Validate(terraform.NewResourceConfigRaw(raw))

			var text string
			for _, d := range diags {
				text += d.Summary + " " + d.Detail + "\n"
			}
			if len(tc.refuse) == 0 {
				if diags.HasError() {
					t.Fatalf("this configuration must validate; got: %s", text)
				}
				return
			}
			if !diags.HasError() {
				t.Fatal("expected a refusal, got none")
			}
			// Pinned as one ordered walk, so the sequence is part of the contract.
			rest := text
			for _, want := range tc.refuse {
				i := strings.Index(rest, want)
				if i < 0 {
					t.Fatalf("the refusal must carry %q, in order; got: %s", want, text)
				}
				rest = rest[i+len(want):]
			}
		})
	}
}

// TestVMwareCloudInitCreateBackstop pins B1 of the 23/11 review: CREATION had no
// refusal of its own, and every plan-time cloud_init rule fails open on a value
// the plan does not know.
//
// The SDK validates a configuration once, with the values the plan holds, and
// skips an unknown one; the collision CustomizeDiff reads through d.Get, where
// an unresolved map reads as EMPTY. Neither runs again at apply. On update that
// costs nothing, because planCloudInitEdits re-applies both rules on resolved
// values before any write — creation had no such copy, merged the two maps,
// gave deploy_options precedence and sent whatever it was handed.
//
// So `cloud_init = { public-keys = tls_private_key.k.public_key_openssh }`
// validated clean and deployed a value the platform stores as &#x2F;, with state
// recording the declared form and the refresh deliberately leaving the key
// alone. Nothing reported it, ever.
//
// Non-complacency: the table carries four configurations that MUST pass (a clean
// map, unrelated deploy_options, a cloud-init key owned only by deploy_options,
// and no cloud_init at all), so a backstop that over-refuses goes red; and the
// wiring subtests drive the real computeVirtualMachineCreate against a server
// that fails the test on ANY request, so "refused before any platform call" is
// asserted rather than assumed.
func TestVMwareCloudInitCreateBackstop(t *testing.T) {
	data := func(t *testing.T, cloudInit, deploy map[string]interface{}) *schema.ResourceData {
		t.Helper()
		raw := map[string]interface{}{
			"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"guest_operating_system_moref": "ubuntu64Guest", "memory": 2147483648, "cpu": 2,
		}
		if cloudInit != nil {
			raw["cloud_init"] = cloudInit
		}
		if deploy != nil {
			raw["deploy_options"] = deploy
		}
		return schema.TestResourceDataRaw(t, resourceVirtualMachine().Schema, raw)
	}

	for _, tc := range []struct {
		name      string
		cloudInit map[string]interface{}
		deploy    map[string]interface{}
		want      []string // every fragment must appear in the diagnostics
		wantCount int      // 0 means "exactly len(want) > 0 is not asserted"
	}{
		{
			// The review's own scenario, with the value resolved as it is at apply.
			name:      "a value the platform would escape is refused",
			cloudInit: map[string]interface{}{"public-keys": "ssh-ed25519 AAAA/BBB"},
			want:      []string{"public-keys", "contains '/' at offset 16", "stored faithfully"},
			wantCount: 1,
		},
		{
			name:      "a value the platform would trim is refused",
			cloudInit: map[string]interface{}{"hostname": " web01 "},
			want:      []string{"hostname", "leading or trailing whitespace"},
			wantCount: 1,
		},
		{
			name:      "an empty value is refused",
			cloudInit: map[string]interface{}{"hostname": ""},
			want:      []string{"hostname", "is empty"},
			wantCount: 1,
		},
		{
			// The key rule fails open on an unknown map too, so the backstop
			// re-applies validateCloudInitMap whole rather than its value half.
			name:      "a key the platform does not know is refused",
			cloudInit: map[string]interface{}{"not-a-cloud-init-key": "x"},
			want:      []string{"not allowed for cloud-init"},
			wantCount: 1,
		},
		{
			name:      "a key declared in deploy_options as well is refused",
			cloudInit: map[string]interface{}{"user-data": "bmV3"},
			deploy:    map[string]interface{}{"user-data": "owned-by-deploy"},
			want:      []string{"user-data declared in both cloud_init and deploy_options"},
			wantCount: 1,
		},
		{
			// Map iteration order is random; the diagnostic must not be.
			name:      "every shared key is named, in a stable order",
			cloudInit: map[string]interface{}{"user-data": "bmV3", "hostname": "web01"},
			deploy:    map[string]interface{}{"user-data": "a", "hostname": "b", "disk_provisioning_type": "thin"},
			want:      []string{"hostname, user-data declared in both"},
			wantCount: 1,
		},
		{
			// Both halves report, so neither hides the other.
			name:      "a value fault and a collision are both reported",
			cloudInit: map[string]interface{}{"public-keys": "ssh-ed25519 AAAA/BBB"},
			deploy:    map[string]interface{}{"public-keys": "owned-by-deploy"},
			want:      []string{"contains '/' at offset 16", "declared in both cloud_init and deploy_options"},
			wantCount: 2,
		},
		{
			name:      "a transmittable map passes",
			cloudInit: map[string]interface{}{"user-data": "bmV3", "hostname": "web01"},
		},
		{
			name:      "unrelated deploy_options alongside cloud_init pass",
			cloudInit: map[string]interface{}{"user-data": "bmV3"},
			deploy:    map[string]interface{}{"disk_provisioning_type": "thin"},
		},
		{
			name:      "a cloud-init key owned only by deploy_options passes",
			cloudInit: map[string]interface{}{"user-data": "bmV3"},
			deploy:    map[string]interface{}{"hostname": "web01"},
		},
		{
			name:   "no cloud_init at all passes",
			deploy: map[string]interface{}{"hostname": "web01"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			diags := cloudInitCreateBackstop(data(t, tc.cloudInit, tc.deploy))
			if len(tc.want) == 0 {
				if diags.HasError() {
					t.Fatalf("a valid configuration must not be refused at creation: %v", diags)
				}
				return
			}
			if !diags.HasError() {
				t.Fatalf("creation accepted a configuration the plan refuses: %#v", tc.cloudInit)
			}
			if tc.wantCount > 0 && len(diags) != tc.wantCount {
				t.Fatalf("got %d diagnostics, want %d: %v", len(diags), tc.wantCount, diags)
			}
			var got strings.Builder
			for _, d := range diags {
				fmt.Fprintf(&got, "%s|%s\n", d.Summary, d.Detail)
			}
			for _, want := range tc.want {
				if !strings.Contains(got.String(), want) {
					t.Fatalf("diagnostics must contain %q; got:\n%s", want, got.String())
				}
			}
		})
	}

	// The wiring. The behavioural table above would stay green if the backstop
	// were never called, which is precisely the defect: the refusal has to sit
	// in computeVirtualMachineCreate, ahead of every platform call.
	for _, tc := range []struct {
		name      string
		cloudInit map[string]interface{}
		deploy    map[string]interface{}
		want      string
	}{
		{
			name:      "a value the platform would escape never reaches the platform",
			cloudInit: map[string]interface{}{"public-keys": "ssh-ed25519 AAAA/BBB"},
			want:      "stored faithfully",
		},
		{
			name:      "a shared key never reaches the platform",
			cloudInit: map[string]interface{}{"user-data": "bmV3"},
			deploy:    map[string]interface{}{"user-data": "owned-by-deploy"},
			want:      "declared in both cloud_init and deploy_options",
		},
	} {
		t.Run("create: "+tc.name, func(t *testing.T) {
			c := newAssignTestClient(t, func(w http.ResponseWriter, q *http.Request) {
				t.Errorf("a refused cloud_init must reach NO platform call; got %s %s", q.Method, q.URL.Path)
				w.WriteHeader(500)
			})
			diags := computeVirtualMachineCreate(context.Background(), data(t, tc.cloudInit, tc.deploy), c)
			if !diags.HasError() {
				t.Fatal("creation was not refused")
			}
			var got strings.Builder
			for _, d := range diags {
				fmt.Fprintf(&got, "%s|%s\n", d.Summary, d.Detail)
			}
			if !strings.Contains(got.String(), tc.want) {
				t.Fatalf("creation must fail with the cloud_init refusal; got:\n%s", got.String())
			}
		})
	}

	// And the other direction: a transmittable map must NOT be refused here, so
	// the create goes on and reaches the platform. Without this, a backstop that
	// refuses everything would pass every assertion above.
	t.Run("create: a transmittable map is attempted", func(t *testing.T) {
		reached := 0
		c := newAssignTestClient(t, func(w http.ResponseWriter, q *http.Request) {
			reached++
			w.WriteHeader(500)
		})
		d := data(t, map[string]interface{}{"user-data": "bmV3"}, map[string]interface{}{"hostname": "web01"})
		diags := computeVirtualMachineCreate(context.Background(), d, c)
		if reached == 0 {
			t.Fatal("a transmittable cloud_init was refused before any platform call")
		}
		if !diags.HasError() {
			t.Fatal("the stub server fails every call, so the create must still report an error")
		}
	})
}
