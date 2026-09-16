package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestCreateOpenIaasVirtualMachineNetworkAdapters pins the WIRE shape of the
// VM-create body's networkAdapters[] entries.
//
// The `ipAddress` field is the one issue #376 recorded as ABSENT from the
// VM-create schema — the reason the inline os_network_adapter block could not
// choose a VPC static IP. It is verified live on the DEV broker to be honoured,
// and these assertions are what keep it on the wire: a wrong or removed json tag
// would silently downgrade a deliberately chosen address to an auto-assigned one,
// with no error anywhere.
func TestCreateOpenIaasVirtualMachineNetworkAdapters(t *testing.T) {
	ctx := context.Background()

	newClient := func(t *testing.T, body *map[string]any) *Client {
		t.Helper()
		return newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/compute/v1/open_iaas/virtual_machines") {
				_ = json.NewDecoder(r.Body).Decode(body)
				w.Header().Set("Location", "/activity/v1/activities/act-1")
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})
	}

	adapterOf := func(t *testing.T, body map[string]any) map[string]any {
		t.Helper()
		adapters, ok := body["networkAdapters"].([]any)
		if !ok || len(adapters) != 1 {
			t.Fatalf("networkAdapters not encoded as a 1-element array: %v", body["networkAdapters"])
		}
		nic, ok := adapters[0].(map[string]any)
		if !ok {
			t.Fatalf("networkAdapters[0] is not an object: %v", adapters[0])
		}
		return nic
	}

	t.Run("a set ipAddress is sent as networkAdapters[].ipAddress", func(t *testing.T) {
		var body map[string]any
		c := newClient(t, &body)
		if _, err := c.Compute().OpenIaaS().VirtualMachine().Create(ctx, &CreateOpenIaasVirtualMachineRequest{
			Name:       "vm",
			TemplateID: "tpl-1",
			CPU:        1,
			Memory:     2147483648,
			NetworkAdapters: []OSNetworkAdapter{
				{NetworkID: "net-vpc", IPAddress: "10.0.5.240"},
			},
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
		nic := adapterOf(t, body)
		if nic["networkId"] != "net-vpc" {
			t.Fatalf("networkId = %v, want net-vpc", nic["networkId"])
		}
		if nic["ipAddress"] != "10.0.5.240" {
			t.Fatalf("ipAddress = %v, want 10.0.5.240 — the chosen VPC static IP was lost on the wire", nic["ipAddress"])
		}
	})

	t.Run("an unset ipAddress is omitted entirely", func(t *testing.T) {
		var body map[string]any
		c := newClient(t, &body)
		if _, err := c.Compute().OpenIaaS().VirtualMachine().Create(ctx, &CreateOpenIaasVirtualMachineRequest{
			Name:            "vm",
			TemplateID:      "tpl-1",
			CPU:             1,
			Memory:          2147483648,
			NetworkAdapters: []OSNetworkAdapter{{NetworkID: "net-pb"}},
		}); err != nil {
			t.Fatalf("create: %v", err)
		}
		nic := adapterOf(t, body)
		// omitempty matters here: sending an empty ipAddress on a plain network
		// would hand the broker a value it has no meaning for.
		if _, present := nic["ipAddress"]; present {
			t.Fatalf("an unset ipAddress must be omitted, got %v", nic["ipAddress"])
		}
		if _, present := nic["mac"]; present {
			t.Fatalf("an unset mac must be omitted, got %v", nic["mac"])
		}
	})
}
