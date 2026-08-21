package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

// TestDeployOpenIaasItemNetworkData pins the WIRE shape of the marketplace
// open_iaas deploy body's networkData[] entries.
//
// Why this test carries real weight. The provider used to REFUSE ip_address on the
// marketplace path, on the grounds that NetworkDataMapping had no such field and
// the route had never been measured. The refusal is now lifted: the marketplace
// module maintainer confirmed the deploy endpoint does accept an ipAddress even
// though the published swagger omits it — the same swagger-versus-reality gap #376
// found on the VM-create route.
//
// That makes the json tag the single point of failure. If the name is wrong, or the
// field is dropped, the platform receives a body with no address, assigns one of
// its own, and answers 201: a deliberately chosen VPC static IP is downgraded to an
// arbitrary one with NO error anywhere, in state or on the platform. That is exactly
// the silent-drop the old refusal existed to prevent, and these assertions are what
// replace it.
func TestDeployOpenIaasItemNetworkData(t *testing.T) {
	ctx := context.Background()

	newClient := func(t *testing.T, body *map[string]any) *Client {
		t.Helper()
		return newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/marketplace/v1/items/open_iaas/deploy") {
				_ = json.NewDecoder(r.Body).Decode(body)
				w.Header().Set("Location", "/activity/v1/activities/act-1")
				w.WriteHeader(http.StatusCreated)
				return
			}
			w.WriteHeader(http.StatusNotFound)
		})
	}

	entryOf := func(t *testing.T, body map[string]any) map[string]any {
		t.Helper()
		nd, ok := body["networkData"].([]any)
		if !ok || len(nd) != 1 {
			t.Fatalf("networkData not encoded as a 1-element array: %v", body["networkData"])
		}
		e, ok := nd[0].(map[string]any)
		if !ok {
			t.Fatalf("networkData[0] is not an object: %v", nd[0])
		}
		return e
	}

	t.Run("a set IPAddress is sent as networkData[].ipAddress", func(t *testing.T) {
		var body map[string]any
		c := newClient(t, &body)
		if _, err := c.Marketplace().Item().DeployOpenIaasItem(ctx, &MarketplaceOpenIaasDeployementRequest{
			ID:                  "item-1",
			Name:                "vm",
			StorageRepositoryID: "sr-1",
			NetworkData: []NetworkDataMapping{{
				NetworkAdapterName:   "VIF #0",
				SourceNetworkName:    "PACKER",
				DestinationNetworkId: "net-vpc",
				IPAddress:            "10.0.5.120",
			}},
		}); err != nil {
			t.Fatalf("deploy: %v", err)
		}
		e := entryOf(t, body)
		if e["ipAddress"] != "10.0.5.120" {
			t.Fatalf("ipAddress = %v, want 10.0.5.120 — the chosen VPC static IP was lost on the wire, so the platform would silently assign a different address", e["ipAddress"])
		}
		// The mapping keys must survive alongside it: sending the address while
		// losing the destination network would attach the adapter to the wrong
		// network, which is worse than losing the address.
		if e["destinationNetworkId"] != "net-vpc" {
			t.Fatalf("destinationNetworkId = %v, want net-vpc", e["destinationNetworkId"])
		}
		if e["networkAdapterName"] != "VIF #0" {
			t.Fatalf("networkAdapterName = %v, want \"VIF #0\"", e["networkAdapterName"])
		}
		if e["sourceNetworkName"] != "PACKER" {
			t.Fatalf("sourceNetworkName = %v, want PACKER", e["sourceNetworkName"])
		}
		if body["storageRepositoryId"] != "sr-1" {
			t.Fatalf("storageRepositoryId = %v, want sr-1", body["storageRepositoryId"])
		}
	})

	t.Run("an unset IPAddress is omitted entirely", func(t *testing.T) {
		var body map[string]any
		c := newClient(t, &body)
		if _, err := c.Marketplace().Item().DeployOpenIaasItem(ctx, &MarketplaceOpenIaasDeployementRequest{
			ID:                  "item-1",
			Name:                "vm",
			StorageRepositoryID: "sr-1",
			NetworkData:         []NetworkDataMapping{{NetworkAdapterName: "VIF #0", DestinationNetworkId: "net-pb"}},
		}); err != nil {
			t.Fatalf("deploy: %v", err)
		}
		e := entryOf(t, body)
		// omitempty matters here: on a plain (non-VPC) network the platform has no
		// meaning for an empty address, and the provider must not invent one.
		if _, present := e["ipAddress"]; present {
			t.Fatalf("an unset ipAddress must be omitted, got %v", e["ipAddress"])
		}
		if _, present := e["sourceNetworkName"]; present {
			t.Fatalf("an unset sourceNetworkName must be omitted, got %v", e["sourceNetworkName"])
		}
	})
}
