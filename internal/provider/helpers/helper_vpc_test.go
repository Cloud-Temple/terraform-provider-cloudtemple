package helpers

import (
	"sort"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// These tests pin the VPC flatten helpers (FlattenVPC, FlattenPrivateNetwork,
// FlattenStaticIP, FlattenFloatingIP) NON-COMPLACENTLY:
//
//   - exact emitted key set vs the datasource schema (the #243 class: an
//     undeclared key makes d.Set fail and breaks the whole datasource read);
//   - nullable API fields collapse to "" (never a panic, never a dropped key);
//   - nested objects (vpc, virtualMachine, networkAdapter, floatingIp,
//     staticIp, privateNetwork) are mapped to the correct flat attribute.
//
// Each test was mutation-proven during development: injecting the historical
// bug class (emit an undeclared key / wrong nested field / drop a nullable
// guard) turns the relevant assertion RED. The mutations are documented inline.

// keysOf returns the sorted key set of a flatten output, for exact-set asserts.
func keysOf(m map[string]interface{}) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func assertKeySet(t *testing.T, name string, got map[string]interface{}, want []string) {
	t.Helper()
	sort.Strings(want)
	gotKeys := keysOf(got)
	if len(gotKeys) != len(want) {
		t.Fatalf("%s emits %d keys %v, want %d keys %v", name, len(gotKeys), gotKeys, len(want), want)
	}
	for i := range want {
		if gotKeys[i] != want[i] {
			t.Fatalf("%s key set = %v, want %v (an undeclared key breaks the datasource read, #243)", name, gotKeys, want)
		}
	}
}

// --- FlattenVPC -----------------------------------------------------------

// vpcSchemaKeys mirrors the cloudtemple_vpc_vpc(s) datasource computed
// attributes the flatten must fill (id is emitted by the helper itself).
var vpcSchemaKeys = []string{
	"id", "name", "internet_ip",
	"private_network_count", "static_ip_count", "floating_ip_count",
}

func TestFlattenVPCKeySetAndValues(t *testing.T) {
	internet := "203.0.113.1"
	vpc := &client.VPC{
		ID:                  "960fb87a-0e84-4e6e-a6e6-d688dfefe6a8",
		Name:                "VPC 1",
		InternetIP:          &internet,
		PrivateNetworkCount: 3,
		StaticIPCount:       12,
		FloatingIPCount:     2,
	}

	got := FlattenVPC(vpc)

	// Exact key set vs schema. Mutation proof: adding result["extra"]=... in
	// FlattenVPC turns this RED (and would break d.Set on the real datasource).
	assertKeySet(t, "FlattenVPC", got, vpcSchemaKeys)

	assertEq(t, "id", got["id"], "960fb87a-0e84-4e6e-a6e6-d688dfefe6a8")
	assertEq(t, "name", got["name"], "VPC 1")
	// Mutation proof: dropping the *InternetIP deref (emitting the pointer)
	// turns this RED with a type mismatch.
	assertEq(t, "internet_ip", got["internet_ip"], "203.0.113.1")
	assertEq(t, "private_network_count", got["private_network_count"], 3)
	assertEq(t, "static_ip_count", got["static_ip_count"], 12)
	assertEq(t, "floating_ip_count", got["floating_ip_count"], 2)
}

// TestFlattenVPCNullInternetIP pins that a null internetIp collapses to "" and
// the key is still emitted. Mutation proof: removing the nil guard in
// FlattenVPC panics here (nil pointer deref).
func TestFlattenVPCNullInternetIP(t *testing.T) {
	got := FlattenVPC(&client.VPC{ID: "x", InternetIP: nil})
	assertEq(t, "internet_ip", got["internet_ip"], "")
	if _, ok := got["internet_ip"]; !ok {
		t.Errorf("FlattenVPC dropped internet_ip on a null value; the Computed attribute must always be set")
	}
}

// --- FlattenPrivateNetwork ------------------------------------------------

var privateNetworkSchemaKeys = []string{
	"id", "name", "ip_address", "vlan_id", "static_ip_count", "vpc_id",
}

func TestFlattenPrivateNetworkKeySetAndValues(t *testing.T) {
	name := "Production network"
	pn := &client.PrivateNetwork{
		ID:            "c229c411-ac30-4caa-9c67-70d4c230d0ee",
		IPAddress:     "192.168.1.0/24",
		Name:          &name,
		VlanID:        100,
		StaticIPCount: 5,
		VPC:           client.BaseObject{ID: "aff7e62f-c603-419d-ae0c-03441abf0655", Name: "Production VPC"},
	}

	got := FlattenPrivateNetwork(pn)

	assertKeySet(t, "FlattenPrivateNetwork", got, privateNetworkSchemaKeys)

	assertEq(t, "id", got["id"], "c229c411-ac30-4caa-9c67-70d4c230d0ee")
	assertEq(t, "name", got["name"], "Production network")
	// ip_address is the CIDR (PrivateNetwork.ipAddress per swagger). Mutation
	// proof: mapping network.Name into ip_address turns this RED.
	assertEq(t, "ip_address", got["ip_address"], "192.168.1.0/24")
	assertEq(t, "vlan_id", got["vlan_id"], 100)
	assertEq(t, "static_ip_count", got["static_ip_count"], 5)
	// The associated VPC is flattened to its id. Mutation proof: emitting
	// network.VPC.Name here turns this RED.
	assertEq(t, "vpc_id", got["vpc_id"], "aff7e62f-c603-419d-ae0c-03441abf0655")
}

func TestFlattenPrivateNetworkNullName(t *testing.T) {
	got := FlattenPrivateNetwork(&client.PrivateNetwork{ID: "x", Name: nil})
	assertEq(t, "name", got["name"], "")
}

// --- FlattenStaticIP ------------------------------------------------------

var staticIPSchemaKeys = []string{
	"id", "ip_address", "mac_address", "source", "resource_description",
	"virtual_machine_id", "network_adapter_id",
	"floating_ip_id", "floating_ip_address",
	"vpc_id", "private_network_id",
}

func TestFlattenStaticIPKeySetAndValues(t *testing.T) {
	desc := "Web server production"
	si := &client.StaticIP{
		ID:                  "4f759498-05ff-42ec-b40e-90c1c9c77541",
		IPAddress:           "10.0.1.50",
		MacAddress:          "00:50:56:ab:cd:ef",
		Source:              "xoa",
		ResourceDescription: &desc,
		VirtualMachine:      &client.BaseObject{ID: "fc3a1ed8-737f-4667-acaa-320d3f523b6f", Name: "VM-001"},
		NetworkAdapter:      &client.BaseObject{ID: "3170b1ef-b4e3-4b8b-a4d7-85e9689ee442", Name: "Network adapter 1"},
		FloatingIP:          &client.StaticIPFloatingIP{ID: "8a9f3c5d-2e4b-4a7f-9c8d-1e5f6b7a9c2d", IPAddress: "198.51.100.61"},
		VPC:                 client.BaseObject{ID: "a1b2c3d4-e5f6-7890-1234-567890abcdef", Name: "VPC-001"},
		PrivateNetwork:      client.BaseObject{ID: "b2c3d4e5-f678-9012-3456-7890abcdef12", Name: "Private Network 1"},
	}

	got := FlattenStaticIP(si)

	assertKeySet(t, "FlattenStaticIP", got, staticIPSchemaKeys)

	assertEq(t, "id", got["id"], "4f759498-05ff-42ec-b40e-90c1c9c77541")
	assertEq(t, "ip_address", got["ip_address"], "10.0.1.50")
	assertEq(t, "mac_address", got["mac_address"], "00:50:56:ab:cd:ef")
	assertEq(t, "source", got["source"], "xoa")
	assertEq(t, "resource_description", got["resource_description"], "Web server production")
	assertEq(t, "virtual_machine_id", got["virtual_machine_id"], "fc3a1ed8-737f-4667-acaa-320d3f523b6f")
	assertEq(t, "network_adapter_id", got["network_adapter_id"], "3170b1ef-b4e3-4b8b-a4d7-85e9689ee442")
	// floatingIp.ipAddress (NOT staticIp.address) is the nested address name for
	// a StaticIp. Mutation proof: reading a wrong nested field turns this RED.
	assertEq(t, "floating_ip_id", got["floating_ip_id"], "8a9f3c5d-2e4b-4a7f-9c8d-1e5f6b7a9c2d")
	assertEq(t, "floating_ip_address", got["floating_ip_address"], "198.51.100.61")
	assertEq(t, "vpc_id", got["vpc_id"], "a1b2c3d4-e5f6-7890-1234-567890abcdef")
	assertEq(t, "private_network_id", got["private_network_id"], "b2c3d4e5-f678-9012-3456-7890abcdef12")
}

// TestFlattenStaticIPNullAssociations pins that every nullable association
// collapses to "" and that ALL keys are still emitted (the schema declares them
// Computed, so a dropped key on an unbound static IP would break the read).
// Mutation proof: removing any nil guard panics here.
func TestFlattenStaticIPNullAssociations(t *testing.T) {
	got := FlattenStaticIP(&client.StaticIP{
		ID:             "x",
		Source:         "vmware",
		VirtualMachine: nil,
		NetworkAdapter: nil,
		FloatingIP:     nil,
		// ResourceDescription nil.
	})
	assertKeySet(t, "FlattenStaticIP", got, staticIPSchemaKeys)
	assertEq(t, "virtual_machine_id", got["virtual_machine_id"], "")
	assertEq(t, "network_adapter_id", got["network_adapter_id"], "")
	assertEq(t, "resource_description", got["resource_description"], "")
	assertEq(t, "floating_ip_id", got["floating_ip_id"], "")
	assertEq(t, "floating_ip_address", got["floating_ip_address"], "")
}

// --- FlattenFloatingIP ----------------------------------------------------

var floatingIPSchemaKeys = []string{
	"id", "ip_address", "description",
	"static_ip_id", "static_ip_address",
	"vpc_id", "private_network_id",
}

func TestFlattenFloatingIPKeySetAndValues(t *testing.T) {
	fip := &client.FloatingIP{
		ID:             "2a1c9d77-db89-4ab8-abba-48852d750df8",
		IPAddress:      "198.51.100.61",
		Description:    "Floating IP for production",
		StaticIP:       &client.FloatingIPStaticIP{ID: "f8a7b6c5-4d3e-2a1b-9c8d-7e6f5a4b3c2d", Address: "10.0.1.5"},
		VPC:            &client.BaseObject{ID: "39ea5bbe-50ff-49b1-82b7-a6857b9aea4c", Name: "VPC Production"},
		PrivateNetwork: &client.BaseObject{ID: "c8f4d8e2-3a1b-4f5c-9d7e-6b8a9c0d1e2f", Name: "Private Network 1"},
	}

	got := FlattenFloatingIP(fip)

	assertKeySet(t, "FlattenFloatingIP", got, floatingIPSchemaKeys)

	assertEq(t, "id", got["id"], "2a1c9d77-db89-4ab8-abba-48852d750df8")
	assertEq(t, "ip_address", got["ip_address"], "198.51.100.61")
	assertEq(t, "description", got["description"], "Floating IP for production")
	assertEq(t, "static_ip_id", got["static_ip_id"], "f8a7b6c5-4d3e-2a1b-9c8d-7e6f5a4b3c2d")
	// staticIp.address (NOT ipAddress) is the nested address name for a
	// FloatingIp. Mutation proof: reading a wrong nested field turns this RED.
	assertEq(t, "static_ip_address", got["static_ip_address"], "10.0.1.5")
	assertEq(t, "vpc_id", got["vpc_id"], "39ea5bbe-50ff-49b1-82b7-a6857b9aea4c")
	assertEq(t, "private_network_id", got["private_network_id"], "c8f4d8e2-3a1b-4f5c-9d7e-6b8a9c0d1e2f")
}

// TestFlattenFloatingIPUnbound pins the unbound case (null staticIp/vpc/
// privateNetwork): all associations collapse to "" while every key stays
// emitted. Mutation proof: removing any nil guard panics here.
func TestFlattenFloatingIPUnbound(t *testing.T) {
	got := FlattenFloatingIP(&client.FloatingIP{
		ID:             "x",
		IPAddress:      "198.51.100.62",
		StaticIP:       nil,
		VPC:            nil,
		PrivateNetwork: nil,
	})
	assertKeySet(t, "FlattenFloatingIP", got, floatingIPSchemaKeys)
	assertEq(t, "static_ip_id", got["static_ip_id"], "")
	assertEq(t, "static_ip_address", got["static_ip_address"], "")
	assertEq(t, "vpc_id", got["vpc_id"], "")
	assertEq(t, "private_network_id", got["private_network_id"], "")
}
