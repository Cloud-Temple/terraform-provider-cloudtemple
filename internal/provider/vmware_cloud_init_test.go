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
	for _, mode := range []string{"read403", "read206", "missing-list", "null-list", "wrong-id", "duplicate-key", "duplicate-id", "running-no-optin", "power-unknown", "disabled", "missing-managed", "drift", "unsafe-value", "empty-value", "deploy-collision", "mixed-change", "post-lost", "post-delayed", "wait-add", "read-add", "wrong-clone", "collateral-change", "concurrent-before-delete", "delete-lost", "wait-delete", "read-after-delete", "final-vapp-check", "verified-then-read-fails", "cancel", "id-missing", "id-null", "id-number", "later-invalid"} {
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
			if mode == "deploy-collision" {
				state.Attributes["deploy_options.%"] = "1"
				state.Attributes["deploy_options.user-data"] = "legacy"
				cfg.Raw["deploy_options"] = map[string]interface{}{"user-data": "legacy"}
				cfg.Config = cfg.Raw
			}
			if mode == "running-no-optin" {
				cfg.Raw["power_state"] = "on"
				cfg.Config = cfg.Raw
			}
			if mode == "mixed-change" {
				cfg.Raw["name"] = "new-name"
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
			// would be permanent (cloud_init is never hydrated by the read).
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
				"power-unknown":    "unsupported power state",
				"running-no-optin": "allow_vm_restart",
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
	t.Run("do not adopt preexisting property", func(t *testing.T) {
		_, err := planCloudInitEdits(snapshot(), nil, map[string]interface{}{"hostname": "after"}, nil)
		if err == nil {
			t.Fatal("adopted unmanaged property")
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
	reads     int    // VM reads taken since the last vApp write
	postOff   int    // reads taken since the shutdown
	secretLoc bool   // return an activity id carrying the secret marker
}

func (f *cloudInitPowerFake) handler(w http.ResponseWriter, q *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	afterShutdown := len(f.actions) == 1 && f.actions[0] == "power:off"
	switch {
	case q.Method == "GET" && q.URL.Path == "/compute/v1/vcenters/virtual_machines/vm-investigation":
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
			"vAppEnabled": true, "vAppProperties": f.props,
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
		{name: "apply-refuses-on-live-authority", statePower: "off", cfgPower: "on", livePower: "running",
			applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk", diagnostic: "allow_vm_restart"},
		// An unsupported live power state is not evidence of "stopped".
		{name: "unsupported-power-state-fails-closed", statePower: "off", cfgPower: "off", livePower: "secret-from-api",
			applyFails: true, actions: "[]", wantIntent: "b2xk", wantLive: "secret-from-api",
			diagnostic: "unsupported power state"},
		// Exempting the two power attributes must not have opened the guard.
		{name: "other-attribute-still-refused", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			otherAttr: "name", applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: `"name"`},
		{name: "other-attribute-tags-still-refused", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			otherAttr: "tags", applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: `"tags"`},
		// An inapplicable configuration must cost NO downtime.
		{name: "invalid-config-refused-before-downtime", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			unsafe: true, applyFails: true, actions: "[]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "cannot update cloud_init"},

		// --- failures around the shutdown -------------------------------------
		// An unconfirmed shutdown must NOT be "compensated" by a power-on that
		// could race a shutdown still settling.
		{name: "unconfirmed-shutdown-fails-closed", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failRead: true, applyFails: true, actions: "[power:off]", wantLive: "stopped", wantIntent: "b2xk",
			diagnostic: "could not be re-read"},
		{name: "shutdown-reported-but-still-running", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			staysOn: true, applyFails: true, actions: "[power:off]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "not stopped after the power-off"},
		{name: "power-off-activity-fails", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failPower: "off", applyFails: true, actions: "[power:off]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "may still be in progress"},
		// A confirmed shutdown followed by a failure that wrote nothing DOES give
		// the machine back — the mirror image of the invariant below.
		{name: "drift-during-shutdown-restores-power", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, applyFails: true, actions: "[power:off power:on]", wantLive: "running", wantIntent: "b2xk",
			diagnostic: "changed while the virtual machine was being powered off"},

		// --- the two load-bearing invariants ----------------------------------
		// A write whose outcome is unknown must never be booted into.
		{name: "write-activity-failure-stays-stopped", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failWrite: "DELETE", applyFails: true, actions: "[power:off POST DELETE]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true},
		// Same, when the HTTP call itself fails: mutated must already be set
		// BEFORE the request, not after a successful return.
		{name: "write-call-failure-stays-stopped", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			failCall: "POST", applyFails: true, actions: "[power:off POST]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true},
		// A VERIFIED transaction whose power settlement fails must KEEP its new
		// intent: rolling it back would make every later apply refuse forever.
		{name: "verified-then-start-fails-keeps-intent", statePower: "off", cfgPower: "on", livePower: "stopped",
			failPower: "on", applyFails: true, actions: "[POST DELETE power:on]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "applied and verified"},
		// A completed power-on activity is not evidence that the machine runs.
		{name: "start-not-confirmed-keeps-intent", statePower: "off", cfgPower: "on", livePower: "stopped",
			onNoOp: true, applyFails: true, actions: "[POST DELETE power:on]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "could not be confirmed powered on"},
		// Settlement must not act on a state it cannot interpret: "not the target"
		// is not "the opposite of the target", and powering on could race a
		// shutdown that has not settled.
		{name: "settlement-refuses-transitional-state", statePower: "off", cfgPower: "on", livePower: "stopped",
			oddLater: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped",
			wantIntent: "bmV3", diagnostic: "unsupported power state, so its power state was left untouched"},
		// The activity id is taken verbatim from a Location header the client does
		// not constrain, so it must never be echoed into a diagnostic.
		{name: "activity-id-is-never-echoed", statePower: "off", cfgPower: "off", livePower: "stopped",
			secretLoc: true, applyFails: true, actions: "[POST]", wantLive: "stopped",
			wantIntent: "b2xk", recovery: true, diagnostic: "not reportable"},
		// "This apply wrote nothing" is NOT "the machine is safe to start". If
		// another writer leaves a duplicate id behind, the provider must not be
		// the actor that boots a guest into it — not even to undo its own
		// shutdown, and not even though the re-plan already named the problem.
		{name: "duplicate-after-shutdown-stays-off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			dupAfter: true, applyFails: true, actions: "[power:off]", wantLive: "stopped", wantIntent: "b2xk",
			diagnostic: "duplicate entries for a managed cloud_init key"},
		// Same rule once the transaction is verified: settlement sees the
		// duplicate in the very snapshot it reads, so it must not power on.
		{name: "duplicate-before-settlement-stays-off", statePower: "off", cfgPower: "on", livePower: "stopped",
			dupSettle: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped", wantIntent: "bmV3",
			diagnostic: "duplicate entries for a managed cloud_init key"},
		// Not only duplicates: any divergence from the snapshot this apply proved
		// means another writer is active, and the guest must not boot into it.
		{name: "vapp-edited-before-settlement-stays-off", statePower: "off", cfgPower: "on", livePower: "stopped",
			modSettle: true, applyFails: true, actions: "[POST DELETE]", wantLive: "stopped", wantIntent: "bmV3",
			diagnostic: "no longer matches the state this apply verified"},
		// A transitional state is not a stopped one, on the compensation path too.
		{name: "transitional-before-restore-stays-off", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, oddRestor: true, applyFails: true, actions: "[power:off]", wantLive: "stopped",
			wantIntent: "b2xk", diagnostic: "not observably stopped"},
		// The same discipline on the compensation path: a power-on that does not
		// take effect must be reported as unconfirmed, not as a restored machine.
		{name: "restore-not-confirmed-is-reported", statePower: "on", cfgPower: "on", allow: true, livePower: "running",
			drift: true, onNoOp: true, applyFails: true, actions: "[power:off power:on]", wantLive: "stopped",
			wantIntent: "b2xk", diagnostic: "could not be confirmed running again"},
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
			if tc.applyFails {
				return
			}
			if next.Attributes["power_state"] != tc.wantState {
				t.Fatalf("recorded power_state = %q, want %q", next.Attributes["power_state"], tc.wantState)
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
// PERMANENT rather than merely untidy: cloud_init is never hydrated from the
// machine, so the next plan would ask to add a key the vApp already has (which
// the planner refuses to adopt) or to restore one it no longer has (which the
// planner refuses as lost), for ever. Each shape is exercised separately
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

// TestVMwareCloudInitPlanGuardUnknowns pins the plan-time guard's edges. It must
// refuse exactly one situation and fail OPEN on everything it cannot decide:
// a value Terraform has not resolved yet must never turn into a refusal.
func TestVMwareCloudInitPlanGuardUnknowns(t *testing.T) {
	const unknown = "74D93920-ED26-11E3-AC10-0800200C9A66" // hcl2shim.UnknownVariableValue
	base := func() (*terraform.InstanceState, map[string]interface{}) {
		return &terraform.InstanceState{ID: "vm-1", Attributes: map[string]string{
			"id": "vm-1", "name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "on", "cloud_init.%": "1", "cloud_init.user-data": "b2xk",
		}}, map[string]interface{}{
			"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc",
			"power_state": "on", "cloud_init": map[string]interface{}{"user-data": "bmV3"},
		}
	}
	for _, tc := range []struct {
		name   string
		mutate func(*terraform.InstanceState, map[string]interface{})
		refuse bool
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
	} {
		t.Run(tc.name, func(t *testing.T) {
			st, raw := base()
			if tc.mutate != nil {
				tc.mutate(st, raw)
			}
			_, err := resourceVirtualMachine().Diff(context.Background(), st, terraform.NewResourceConfigRaw(raw), nil)
			switch {
			case tc.refuse && (err == nil || !strings.Contains(err.Error(), "allow_vm_restart")):
				t.Fatalf("expected the cloud_init power refusal, got: %v", err)
			case !tc.refuse && err != nil:
				t.Fatalf("the guard must fail open here, got: %v", err)
			}
		})
	}
}
