package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
)

// newVPCTestClient wires a Client to a stub HTTP server and pre-seeds a
// far-future JWT so JWT() never hits the network. The VPC tests exercise the
// path construction, the query-param filters, the swagger field mapping and
// the not-found handling — not the auth plumbing.
func newVPCTestClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)

	c, err := NewClient(&Config{Address: srv.URL})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.SavedToken = &jwt.Token{
		Claims: jwt.MapClaims{"exp": float64(time.Now().Add(time.Hour).Unix())},
	}
	return c
}

// --- VPC ------------------------------------------------------------------

func TestVPCVPCListAndRead(t *testing.T) {
	ctx := context.Background()

	t.Run("List hits /vpc/v1/vpc and decodes a nullable internetIp", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet || r.URL.Path != "/vpc/v1/vpc" {
				t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":"v1","name":"VPC 1","internetIp":"203.0.113.1","privateNetworkCount":3,"staticIpCount":12,"floatingIpCount":2},
				{"id":"v2","name":"VPC 2","internetIp":null,"privateNetworkCount":0,"staticIpCount":0,"floatingIpCount":0}
			]`))
		})
		vpcs, err := c.VPC().VPC().List(ctx)
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(vpcs) != 2 {
			t.Fatalf("got %d vpcs, want 2", len(vpcs))
		}
		if vpcs[0].InternetIP == nil || *vpcs[0].InternetIP != "203.0.113.1" {
			t.Errorf("vpc[0].InternetIP = %v, want 203.0.113.1", vpcs[0].InternetIP)
		}
		if vpcs[0].PrivateNetworkCount != 3 || vpcs[0].StaticIPCount != 12 || vpcs[0].FloatingIPCount != 2 {
			t.Errorf("vpc[0] counts mismatch: %+v", vpcs[0])
		}
		// A null internetIp must decode to a nil pointer, not "".
		if vpcs[1].InternetIP != nil {
			t.Errorf("vpc[1].InternetIP = %v, want nil for a null value", *vpcs[1].InternetIP)
		}
	})

	t.Run("Read hits /vpc/v1/vpc/{id}", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/vpc/v1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"v1","name":"VPC 1","internetIp":"203.0.113.1","privateNetworkCount":1,"staticIpCount":2,"floatingIpCount":3}`))
		})
		vpc, err := c.VPC().VPC().Read(ctx, "v1")
		if err != nil {
			t.Fatalf("Read: %v", err)
		}
		if vpc == nil || vpc.ID != "v1" {
			t.Fatalf("unexpected vpc: %+v", vpc)
		}
	})

	t.Run("Read returns (nil,nil) on 404", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		vpc, err := c.VPC().VPC().Read(ctx, "missing")
		if err != nil || vpc != nil {
			t.Fatalf("404 must yield (nil,nil); got vpc=%+v err=%v", vpc, err)
		}
	})
}

// --- PrivateNetwork -------------------------------------------------------

func TestVPCPrivateNetworkListFilterAndCIDR(t *testing.T) {
	ctx := context.Background()

	t.Run("List forwards the vpcId filter and decodes ipAddress as the CIDR", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/private_networks" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("vpcId"); got != "vpc-42" {
				t.Errorf("vpcId filter = %q, want vpc-42", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":"pn1","ipAddress":"192.168.1.0/24","name":"Prod","vlanId":100,"staticIpCount":5,"vpc":{"id":"vpc-42","name":"Prod VPC"}},
				{"id":"pn2","ipAddress":"10.0.0.0/16","name":null,"vlanId":200,"staticIpCount":0,"vpc":{"id":"vpc-42","name":"Prod VPC"}}
			]`))
		})
		pns, err := c.VPC().PrivateNetwork().List(ctx, &PrivateNetworkFilter{VpcID: "vpc-42"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(pns) != 2 {
			t.Fatalf("got %d, want 2", len(pns))
		}
		if pns[0].IPAddress != "192.168.1.0/24" {
			t.Errorf("ipAddress = %q, want the CIDR 192.168.1.0/24", pns[0].IPAddress)
		}
		if pns[0].VPC.ID != "vpc-42" {
			t.Errorf("vpc.id = %q, want vpc-42", pns[0].VPC.ID)
		}
		// A null name decodes to a nil pointer.
		if pns[1].Name != nil {
			t.Errorf("pn[1].Name = %v, want nil for a null value", *pns[1].Name)
		}
	})

	t.Run("List without a filter sends no vpcId param", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if _, ok := r.URL.Query()["vpcId"]; ok {
				t.Errorf("vpcId must be absent when the filter is empty; got %q", r.URL.RawQuery)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[]`))
		})
		if _, err := c.VPC().PrivateNetwork().List(ctx, &PrivateNetworkFilter{}); err != nil {
			t.Fatalf("List: %v", err)
		}
	})
}

// --- StaticIP -------------------------------------------------------------

// TestVPCStaticIPMapping pins the swagger field mapping for a static IP,
// including the nested floatingIp.ipAddress field name (distinct from the
// FloatingIp.staticIp.address used elsewhere) and the source enum.
func TestVPCStaticIPMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("Read maps every nested object and the source enum", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/static_ips/si1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"si1","ipAddress":"10.0.1.50","macAddress":"00:50:56:ab:cd:ef",
				"virtualMachine":{"id":"vm-1","name":"VM-001"},
				"networkAdapter":{"id":"na-1","name":"Network adapter 1"},
				"source":"xoa","resourceDescription":"Web",
				"floatingIp":{"id":"fip-1","ipAddress":"198.51.100.61"},
				"vpc":{"id":"vpc-1","name":"VPC-001"},
				"privateNetwork":{"id":"pn-1","name":"Private Network 1"}
			}`))
		})
		si, err := c.VPC().StaticIP().Read(ctx, "si1")
		if err != nil || si == nil {
			t.Fatalf("Read: si=%+v err=%v", si, err)
		}
		if si.Source != "xoa" {
			t.Errorf("source = %q, want xoa", si.Source)
		}
		if si.VirtualMachine == nil || si.VirtualMachine.ID != "vm-1" {
			t.Errorf("virtualMachine mismapped: %+v", si.VirtualMachine)
		}
		if si.NetworkAdapter == nil || si.NetworkAdapter.ID != "na-1" {
			t.Errorf("networkAdapter mismapped: %+v", si.NetworkAdapter)
		}
		// The nested floating IP address is "ipAddress" for a StaticIp.
		if si.FloatingIP == nil || si.FloatingIP.ID != "fip-1" || si.FloatingIP.IPAddress != "198.51.100.61" {
			t.Errorf("floatingIp mismapped (expected ipAddress field): %+v", si.FloatingIP)
		}
		if si.VPC.ID != "vpc-1" || si.PrivateNetwork.ID != "pn-1" {
			t.Errorf("vpc/privateNetwork mismapped: vpc=%+v pn=%+v", si.VPC, si.PrivateNetwork)
		}
	})

	t.Run("Read decodes null nested objects to nil", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id":"si2","ipAddress":"10.0.1.51","macAddress":"00:50:56:ab:cd:f0",
				"virtualMachine":null,"networkAdapter":null,"source":"vmware",
				"resourceDescription":null,"floatingIp":null,
				"vpc":{"id":"vpc-1","name":"VPC-001"},"privateNetwork":{"id":"pn-1","name":"PN"}
			}`))
		})
		si, err := c.VPC().StaticIP().Read(ctx, "si2")
		if err != nil || si == nil {
			t.Fatalf("Read: si=%+v err=%v", si, err)
		}
		if si.VirtualMachine != nil || si.NetworkAdapter != nil || si.FloatingIP != nil || si.ResourceDescription != nil {
			t.Errorf("null nested objects must decode to nil; got %+v", si)
		}
	})

	t.Run("List hits /private_networks/{id}/static_ips and forwards virtualMachineId", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/private_networks/pn-1/static_ips" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("virtualMachineId"); got != "vm-9" {
				t.Errorf("virtualMachineId filter = %q, want vm-9", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"si1","ipAddress":"10.0.1.50","macAddress":"x","source":"xoa","virtualMachine":null,"networkAdapter":null,"floatingIp":null,"vpc":{"id":"v","name":"n"},"privateNetwork":{"id":"pn-1","name":"n"}}]`))
		})
		sis, err := c.VPC().StaticIP().List(ctx, "pn-1", &StaticIPFilter{VirtualMachineID: "vm-9"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(sis) != 1 || sis[0].ID != "si1" {
			t.Fatalf("unexpected: %+v", sis)
		}
	})

	t.Run("ReadByMAC hits /static_ips/mac/{mac}", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/static_ips/mac/00:50:56:ab:cd:ef" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"si1","ipAddress":"10.0.1.50","macAddress":"00:50:56:ab:cd:ef","source":"xoa","virtualMachine":null,"networkAdapter":null,"floatingIp":null,"vpc":{"id":"v","name":"n"},"privateNetwork":{"id":"p","name":"n"}}`))
		})
		si, err := c.VPC().StaticIP().ReadByMAC(ctx, "00:50:56:ab:cd:ef")
		if err != nil || si == nil || si.ID != "si1" {
			t.Fatalf("ReadByMAC: si=%+v err=%v", si, err)
		}
	})
}

// --- FloatingIP -----------------------------------------------------------

// TestVPCFloatingIPMapping pins the swagger field mapping for a floating IP,
// including the nested staticIp.address field name (distinct from the
// StaticIp.floatingIp.ipAddress used elsewhere) and the nullable associations.
func TestVPCFloatingIPMapping(t *testing.T) {
	ctx := context.Background()

	t.Run("List forwards the vpcId filter and maps a bound floating IP", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/floating_ips" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			if got := r.URL.Query().Get("vpcId"); got != "vpc-7" {
				t.Errorf("vpcId filter = %q, want vpc-7", got)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[
				{"id":"fip-1","ipAddress":"198.51.100.61","description":"prod",
				 "staticIp":{"id":"si-1","address":"10.0.1.5"},
				 "vpc":{"id":"vpc-7","name":"VPC Production"},
				 "privateNetwork":{"id":"pn-1","name":"Private Network 1"}},
				{"id":"fip-2","ipAddress":"198.51.100.62","description":"",
				 "staticIp":null,"vpc":null,"privateNetwork":null}
			]`))
		})
		fips, err := c.VPC().FloatingIP().List(ctx, &FloatingIPFilter{VpcID: "vpc-7"})
		if err != nil {
			t.Fatalf("List: %v", err)
		}
		if len(fips) != 2 {
			t.Fatalf("got %d, want 2", len(fips))
		}
		// The nested static IP address is "address" for a FloatingIp.
		if fips[0].StaticIP == nil || fips[0].StaticIP.ID != "si-1" || fips[0].StaticIP.Address != "10.0.1.5" {
			t.Errorf("staticIp mismapped (expected address field): %+v", fips[0].StaticIP)
		}
		if fips[0].VPC == nil || fips[0].VPC.ID != "vpc-7" {
			t.Errorf("vpc mismapped: %+v", fips[0].VPC)
		}
		if fips[0].PrivateNetwork == nil || fips[0].PrivateNetwork.ID != "pn-1" {
			t.Errorf("privateNetwork mismapped: %+v", fips[0].PrivateNetwork)
		}
		// An unbound floating IP has nil associations.
		if fips[1].StaticIP != nil || fips[1].VPC != nil || fips[1].PrivateNetwork != nil {
			t.Errorf("unbound floating IP must have nil associations; got %+v", fips[1])
		}
	})

	t.Run("Read hits /vpc/v1/floating_ips/{id}", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/vpc/v1/floating_ips/fip-1" {
				t.Errorf("unexpected path: %s", r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"fip-1","ipAddress":"198.51.100.61","description":"prod","staticIp":null,"vpc":null,"privateNetwork":null}`))
		})
		fip, err := c.VPC().FloatingIP().Read(ctx, "fip-1")
		if err != nil || fip == nil || fip.ID != "fip-1" {
			t.Fatalf("Read: fip=%+v err=%v", fip, err)
		}
	})

	t.Run("Read returns (nil,nil) on 404", func(t *testing.T) {
		c := newVPCTestClient(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		})
		fip, err := c.VPC().FloatingIP().Read(ctx, "missing")
		if err != nil || fip != nil {
			t.Fatalf("404 must yield (nil,nil); got fip=%+v err=%v", fip, err)
		}
	})
}
