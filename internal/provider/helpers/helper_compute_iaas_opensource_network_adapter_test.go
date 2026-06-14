package helpers

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// These tests pin FlattenOpenIaaSNetworkAdapter NON-COMPLACENTLY for the
// Volet B additions (#238): the VPC / IP / static-IP attributes are flat
// Computed strings (NOT a nested block), nullable in the API, and must
// collapse to "" without panicking when the adapter is not on a VPC network.
//
// The walker (TestDatasourceFlattenOutputsFitTheirSchemas) only proves the
// EMITTED keys FIT the datasource schemas; it is one-way and datasource-only.
// These tests are the real guard that the new keys are actually EMITTED with
// the correct VALUES, and the resource-side keyset test below covers the
// surface the walker does not.

// openIaaSNetworkAdapterSchemaKeys mirrors the Computed attributes the
// cloudtemple_compute_iaas_opensource_network_adapter(s) datasources and the
// standalone resource expect the flatten to fill.
var openIaaSNetworkAdapterSchemaKeys = []string{
	"name", "internal_id", "virtual_machine_id", "mac_address",
	"mtu", "attached", "tx_checksumming", "network_id", "machine_manager_id",
	"ipv4_address", "ipv6_address",
	"vpc_id", "vpc_name", "private_network_id", "private_network_name",
	"static_ip_address",
}

// TestFlattenOpenIaaSNetworkAdapterOnVPC pins the on-VPC case: every new key is
// emitted with the correct value, and the NON-deprecated nested
// privateNetwork{id,name} is the source (NOT the deprecated top-level fields,
// which the client does not even decode).
//
// Mutation proofs (each turns an assertion RED):
//   - swap vpc_id and private_network_id in the helper -> the two asserts below
//     for those keys fail (their values are distinct on purpose);
//   - drop the static_ip_address mapping -> its assert fails with "";
//   - read adapter.Name into vpc_name -> vpc_name assert fails.
func TestFlattenOpenIaaSNetworkAdapterOnVPC(t *testing.T) {
	adapter := &client.OpenIaaSNetworkAdapter{
		Name:             "VIF #0",
		InternalID:       "8b1c0f3a-1111-2222-3333-444455556666",
		VirtualMachineID: "11111111-2222-3333-4444-555555555555",
		MacAddress:       "0a:1b:2c:3d:4e:5f",
		MTU:              1500,
		Attached:         true,
		TxChecksumming:   true,
		Network:          client.BaseObject{ID: "net-99", Name: "network-99"},
		MachineManager:   client.BaseObject{ID: "mm-7", Name: "XOA"},
		IPv4Address:      "10.0.0.42",
		IPv6Address:      "fe80::42",
		VPC: &client.OpenIaaSNetworkAdapterVPC{
			ID:   "vpc-aaaa",
			Name: "VPC Production",
			PrivateNetwork: struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{ID: "pn-bbbb", Name: "Private Network 1"},
			StaticIPAddress: "10.0.0.200",
		},
	}

	got := FlattenOpenIaaSNetworkAdapter(adapter)

	assertKeySet(t, "FlattenOpenIaaSNetworkAdapter", got, openIaaSNetworkAdapterSchemaKeys)

	assertEq(t, "ipv4_address", got["ipv4_address"], "10.0.0.42")
	assertEq(t, "ipv6_address", got["ipv6_address"], "fe80::42")
	// vpc_id and private_network_id are DISTINCT values: a swap is RED.
	assertEq(t, "vpc_id", got["vpc_id"], "vpc-aaaa")
	assertEq(t, "vpc_name", got["vpc_name"], "VPC Production")
	assertEq(t, "private_network_id", got["private_network_id"], "pn-bbbb")
	assertEq(t, "private_network_name", got["private_network_name"], "Private Network 1")
	assertEq(t, "static_ip_address", got["static_ip_address"], "10.0.0.200")
}

// TestFlattenOpenIaaSNetworkAdapterNoVPC pins the off-VPC case: a nil VPC
// pointer must flatten every vpc_*/private_network_*/static_ip_address key to
// "" WITHOUT panicking, while IP addresses (which exist outside a VPC too)
// still pass through. Mutation proof: removing the nil guard panics here.
func TestFlattenOpenIaaSNetworkAdapterNoVPC(t *testing.T) {
	got := FlattenOpenIaaSNetworkAdapter(&client.OpenIaaSNetworkAdapter{
		Name:        "VIF #1",
		IPv4Address: "192.168.1.10",
		IPv6Address: "",
		VPC:         nil,
	})

	assertKeySet(t, "FlattenOpenIaaSNetworkAdapter", got, openIaaSNetworkAdapterSchemaKeys)

	assertEq(t, "ipv4_address", got["ipv4_address"], "192.168.1.10")
	assertEq(t, "ipv6_address", got["ipv6_address"], "")
	assertEq(t, "vpc_id", got["vpc_id"], "")
	assertEq(t, "vpc_name", got["vpc_name"], "")
	assertEq(t, "private_network_id", got["private_network_id"], "")
	assertEq(t, "private_network_name", got["private_network_name"], "")
	assertEq(t, "static_ip_address", got["static_ip_address"], "")
}
