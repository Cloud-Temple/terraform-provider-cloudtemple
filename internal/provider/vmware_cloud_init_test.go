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
	for _, mode := range []string{"read403", "read206", "missing-list", "null-list", "wrong-id", "duplicate-key", "duplicate-id", "running", "disabled", "missing-managed", "drift", "unsafe-value", "empty-value", "deploy-collision", "mixed-change", "post-lost", "post-delayed", "wait-add", "read-add", "wrong-clone", "collateral-change", "concurrent-before-delete", "delete-lost", "wait-delete", "read-after-delete", "final-read", "final-vapp", "cancel", "id-missing", "id-null", "id-number", "later-invalid"} {
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
					if mode == "read403" || mode == "read-add" && len(writes) == 1 || mode == "read-after-delete" && len(writes) == 2 || mode == "final-read" && readsAfterDelete == 2 || mode == "final-vapp" && readsAfterDelete == 3 {
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
					case "running":
						vm["powerState"] = "running"
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
			if next.ID != state.ID || next.Attributes["cloud_init.user-data"] != "b2xk" {
				t.Fatal("failed apply persisted optimistic cloud_init or lost identity")
			}
			again, err := r.Diff(context.Background(), next, cfg, nil)
			if err != nil {
				t.Fatal(err)
			}
			retryKey := "cloud_init.user-data"
			if strings.HasPrefix(mode, "id-") {
				retryKey = "cloud_init.hostname"
			}
			if again == nil || again.Attributes[retryKey] == nil {
				t.Fatal("failed update suppressed retry intent")
			}
			expectedWrites := 0
			switch mode {
			case "post-lost", "post-delayed", "wait-add", "read-add", "wrong-clone", "collateral-change", "concurrent-before-delete", "cancel":
				expectedWrites = 1
			case "delete-lost", "wait-delete", "read-after-delete", "final-read", "final-vapp":
				expectedWrites = 2
			}
			if len(writes) != expectedWrites {
				t.Fatalf("unexpected mutation count %v, wanted %d", writes, expectedWrites)
			}
			message := fmt.Sprint(diags)
			if strings.Contains(message+logs.String(), "secret-from-api") {
				t.Fatal("diagnostic leaked payload")
			}
			if expectedWrites > 0 && !strings.Contains(message, "Do not reapply") {
				t.Fatal("uncertain mutation needs explicit recovery diagnostic")
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
			state, cfg := cloudInitStateConfig()
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
			cfg = terraform.NewResourceConfigRaw(map[string]interface{}{"name": "investigation", "datacenter_id": "dc", "host_cluster_id": "hc", "guest_operating_system_moref": "ubuntu64Guest", "memory": 2147483648, "cpu": 2, "num_cores_per_socket": 1, "power_state": "off", "cloud_init": desired})
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
