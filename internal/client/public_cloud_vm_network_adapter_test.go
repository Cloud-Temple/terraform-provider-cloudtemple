package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
)

func TestPublicCloudVMNetworkAdapterList(t *testing.T) {
	ctx := context.Background()
	c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/network_adapters" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		// Wrapper {vmId,networks,total} (verified live); ipv4/ipv6 arrive as null.
		_, _ = w.Write([]byte(`{"vmId":"vm-1","total":1,"networks":[
		  {"deviceIndex":0,"networkId":"net-1","networkName":"ahn-1","type":"private_backbone","provisionStatus":"provisioned","macAddress":"12:af:bf:c9:90:b0","ipv4Address":null,"ipv6Address":null,"id":"nic-1"}
		]}`))
	})
	nics, err := c.PublicCloudVM().NetworkAdapter().List(ctx, "vm-1")
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(nics) != 1 {
		t.Fatalf("want 1 nic, got %d", len(nics))
	}
	n := nics[0]
	if n.ID != "nic-1" || n.DeviceIndex != 0 || n.NetworkID != "net-1" || n.NetworkName != "ahn-1" {
		t.Fatalf("nic identity not decoded: %+v", n)
	}
	if n.Type != "private_backbone" || n.ProvisionStatus != "provisioned" || n.MacAddress != "12:af:bf:c9:90:b0" {
		t.Fatalf("nic attrs not decoded: %+v", n)
	}
	// null ipv4/ipv6 must decode to the empty string, never fail the decode.
	if n.IPv4Address != "" || n.IPv6Address != "" {
		t.Fatalf("null ip must decode to empty string, got v4=%q v6=%q", n.IPv4Address, n.IPv6Address)
	}
}

// TestPublicCloudVMNetworkAdapterListStrict pins the E0-9 completeness + structural
// integrity contract: only a complete, self-consistent 200 listing is absence
// evidence. Truncation, a missing total, a mismatched vmId, a malformed entry, a
// 206 and a 403 all fail closed.
func TestPublicCloudVMNetworkAdapterListStrict(t *testing.T) {
	ctx := context.Background()

	strictWith := func(body string, code int) error {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
			_, _ = w.Write([]byte(body))
		})
		_, err := c.PublicCloudVM().NetworkAdapter().ListStrict(ctx, "vm-1")
		return err
	}

	t.Run("complete 200 returns the nics", func(t *testing.T) {
		if err := strictWith(`{"vmId":"vm-1","total":1,"networks":[{"id":"nic-1","deviceIndex":0}]}`, http.StatusOK); err != nil {
			t.Fatalf("complete listing must succeed: %v", err)
		}
	})

	t.Run("truncated (total>len) fails closed", func(t *testing.T) {
		if err := strictWith(`{"vmId":"vm-1","total":5,"networks":[{"id":"nic-1"}]}`, http.StatusOK); err == nil {
			t.Fatal("a truncated listing (total>len) must fail closed")
		}
	})

	t.Run("missing total fails closed", func(t *testing.T) {
		for _, bad := range []string{`{"vmId":"vm-1","networks":[]}`, `{}`} {
			if err := strictWith(bad, http.StatusOK); err == nil {
				t.Fatalf("a listing without a total (%s) must fail closed", bad)
			}
		}
	})

	t.Run("genuine empty (total 0) is accepted", func(t *testing.T) {
		if err := strictWith(`{"vmId":"vm-1","total":0,"networks":[]}`, http.StatusOK); err != nil {
			t.Fatalf("a genuine empty listing (total 0) must be accepted: %v", err)
		}
	})

	t.Run("vmId mismatch fails closed", func(t *testing.T) {
		// A listing scoped to another VM cannot prove this VM's NIC absent.
		if err := strictWith(`{"vmId":"vm-OTHER","total":0,"networks":[]}`, http.StatusOK); err == nil {
			t.Fatal("a wrapper vmId that does not match the requested VM must fail closed")
		}
	})

	t.Run("vmId case-insensitive match is accepted", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"vmId":"VM-1","total":0,"networks":[]}`))
		})
		if _, err := c.PublicCloudVM().NetworkAdapter().ListStrict(ctx, "vm-1"); err != nil {
			t.Fatalf("an upper-case vmId must still match case-insensitively: %v", err)
		}
	})

	t.Run("malformed entry (empty id) fails closed", func(t *testing.T) {
		if err := strictWith(`{"vmId":"vm-1","total":1,"networks":[{"id":"","deviceIndex":0}]}`, http.StatusOK); err == nil {
			t.Fatal("a listing with an empty-id entry must fail closed")
		}
	})

	t.Run("null entry fails closed", func(t *testing.T) {
		// A JSON null decodes to a nil *PublicCloudVMNetworkAdapter; it must be
		// rejected rather than panic or be silently counted toward completeness.
		if err := strictWith(`{"vmId":"vm-1","total":1,"networks":[null]}`, http.StatusOK); err == nil {
			t.Fatal("a listing with a null entry must fail closed")
		}
	})

	t.Run("empty wrapper vmId is accepted", func(t *testing.T) {
		// An absent vmId in the wrapper is allowed (we only reject a NON-empty
		// mismatch); a complete listing must still be usable.
		if err := strictWith(`{"total":0,"networks":[]}`, http.StatusOK); err != nil {
			t.Fatalf("an empty wrapper vmId must be accepted: %v", err)
		}
	})

	t.Run("206 fails closed", func(t *testing.T) {
		if err := strictWith(`{"vmId":"vm-1","total":1,"networks":[{"id":"nic-1"}]}`, http.StatusPartialContent); err == nil {
			t.Fatal("a 206 must fail closed")
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		if err := strictWith(``, http.StatusForbidden); err == nil {
			t.Fatal("a 403 must fail closed")
		}
	})
}

// TestPublicCloudVMNetworkAdapterRead pins that a 404 is absence but a 400 (what
// the live platform returns for an unknown NIC) is a fail-closed error, never a
// silent absent — so the resource can never drop state on a 400.
func TestPublicCloudVMNetworkAdapterRead(t *testing.T) {
	ctx := context.Background()

	t.Run("200 decodes", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/network_adapters/nic-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"nic-1","deviceIndex":1,"networkId":"net-1","type":"vpc"}`))
		})
		nic, err := c.PublicCloudVM().NetworkAdapter().Read(ctx, "vm-1", "nic-1")
		if err != nil || nic == nil || nic.DeviceIndex != 1 || nic.Type != "vpc" {
			t.Fatalf("bad nic: %+v err=%v", nic, err)
		}
	})

	t.Run("404 is absence", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) })
		nic, err := c.PublicCloudVM().NetworkAdapter().Read(ctx, "vm-1", "missing")
		if err != nil || nic != nil {
			t.Fatalf("404 must be (nil,nil), got nic=%+v err=%v", nic, err)
		}
	})

	t.Run("400 fails closed (never a silent absent)", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusBadRequest) })
		nic, err := c.PublicCloudVM().NetworkAdapter().Read(ctx, "vm-1", "unknown")
		if err == nil || nic != nil {
			t.Fatalf("a 400 must fail closed with an error, got nic=%+v err=%v", nic, err)
		}
	})

	t.Run("206 fails closed (only 200 is a found NIC)", func(t *testing.T) {
		// A 206 must never be decoded as a found NIC (it could carry a partial
		// body that bypasses the ListStrict integrity guards).
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte(`{"id":"nic-1"}`))
		})
		nic, err := c.PublicCloudVM().NetworkAdapter().Read(ctx, "vm-1", "nic-1")
		if err == nil || nic != nil {
			t.Fatalf("a 206 must fail closed with an error, got nic=%+v err=%v", nic, err)
		}
	})

	t.Run("403 fails closed", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusForbidden) })
		if _, err := c.PublicCloudVM().NetworkAdapter().Read(ctx, "vm-1", "denied"); err == nil {
			t.Fatal("a 403 must fail closed")
		}
	})
}

func TestPublicCloudVMNetworkAdapterWrites(t *testing.T) {
	ctx := context.Background()

	t.Run("create posts camelCase body, omits empty ipAddress, returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/network_adapters" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-nic")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().NetworkAdapter().Create(ctx, "vm-1", &CreateVMNetworkAdapterRequest{NetworkID: "net-1", DeviceIndex: 0})
		if err != nil || id != "act-nic" {
			t.Fatalf("Create: id=%q err=%v", id, err)
		}
		if body["networkId"] != "net-1" || body["deviceIndex"] != float64(0) {
			t.Fatalf("create body not sent camelCase: %v", body)
		}
		if _, ok := body["ipAddress"]; ok {
			t.Fatalf("empty ipAddress must be omitted: %v", body)
		}
	})

	t.Run("create includes ipAddress when set", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-nic")
			w.WriteHeader(http.StatusCreated)
		})
		_, err := c.PublicCloudVM().NetworkAdapter().Create(ctx, "vm-1", &CreateVMNetworkAdapterRequest{NetworkID: "net-1", DeviceIndex: 1, IPAddress: "10.0.0.5"})
		if err != nil {
			t.Fatalf("Create: %v", err)
		}
		if body["ipAddress"] != "10.0.0.5" || body["deviceIndex"] != float64(1) {
			t.Fatalf("ipAddress/deviceIndex not sent: %v", body)
		}
	})

	t.Run("change-network PATCHes the nic and returns activityId", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/network_adapters/nic-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-patch")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().NetworkAdapter().ChangeNetwork(ctx, "vm-1", "nic-1", &ChangeVMNetworkAdapterRequest{NetworkID: "net-2"})
		if err != nil || id != "act-patch" {
			t.Fatalf("ChangeNetwork: id=%q err=%v", id, err)
		}
		if body["networkId"] != "net-2" {
			t.Fatalf("change-network body not sent: %v", body)
		}
		if _, ok := body["ipAddress"]; ok {
			t.Fatalf("empty ipAddress must be omitted on change-network: %v", body)
		}
	})

	t.Run("change-network includes ipAddress when set", func(t *testing.T) {
		var body map[string]any
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			raw, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(raw, &body)
			w.Header().Set("Location", "act-patch")
			w.WriteHeader(http.StatusCreated)
		})
		_, err := c.PublicCloudVM().NetworkAdapter().ChangeNetwork(ctx, "vm-1", "nic-1", &ChangeVMNetworkAdapterRequest{NetworkID: "net-2", IPAddress: "10.0.0.9"})
		if err != nil {
			t.Fatalf("ChangeNetwork: %v", err)
		}
		if body["networkId"] != "net-2" || body["ipAddress"] != "10.0.0.9" {
			t.Fatalf("change-network body with ip not sent: %v", body)
		}
	})

	t.Run("delete DELETEs the nic and returns activityId", func(t *testing.T) {
		c := newPATTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodDelete || r.URL.Path != "/vm_instances/v1/virtual_machines/vm-1/network_adapters/nic-1" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.Header().Set("Location", "act-del")
			w.WriteHeader(http.StatusCreated)
		})
		id, err := c.PublicCloudVM().NetworkAdapter().Delete(ctx, "vm-1", "nic-1")
		if err != nil || id != "act-del" {
			t.Fatalf("Delete: id=%q err=%v", id, err)
		}
	})
}
