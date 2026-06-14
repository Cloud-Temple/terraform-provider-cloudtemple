package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// The schema-vs-flatten walker (datasource_flatten_schema_test.go) is
// datasource-only: it does NOT cover the standalone network-adapter RESOURCE
// read, which feeds the SAME helpers.FlattenOpenIaaSNetworkAdapter into a
// per-key d.Set loop (resource_compute_iaas_opensource_network_adapter.go).
// A new Computed attribute declared in the resource schema but mis-named in
// the flatten — or emitted by the flatten but not declared — would slip past
// both the walker and the golden gate. This test closes that gap for the
// Volet B additions (#238): it asserts every key the flatten emits on a fully
// VPC-associated adapter is declared in the resource schema AND is accepted by
// the real SDK schema writer.
//
// Mutation proofs:
//   - rename a flatten key (e.g. "vpc_id" -> "vpcid") without touching the
//     schema -> the "not declared" branch fires;
//   - drop one of the 7 new attributes from the resource schema -> same;
//   - emit a wrongly-typed value -> d.Set fails with the offending address.
func TestResourceOpenIaasNetworkAdapterReadFlattenFitsSchema(t *testing.T) {
	res := resourceOpenIaasNetworkAdapter()

	adapter := &client.OpenIaaSNetworkAdapter{
		Name:             "VIF #0",
		InternalID:       "8b1c0f3a-1111-2222-3333-444455556666",
		VirtualMachineID: "11111111-2222-3333-4444-555555555555",
		MacAddress:       "0a:1b:2c:3d:4e:5f",
		MTU:              1500,
		Attached:         true,
		TxChecksumming:   true,
		Network:          client.BaseObject{ID: "net-99"},
		MachineManager:   client.BaseObject{ID: "mm-7"},
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

	flattened := helpers.FlattenOpenIaaSNetworkAdapter(adapter)

	d := schema.TestResourceDataRaw(t, res.Schema, map[string]interface{}{})
	for key, value := range flattened {
		if _, declared := res.Schema[key]; !declared {
			t.Errorf("the resource read flatten emits %q which the resource schema does not declare (would break d.Set)", key)
			continue
		}
		if err := d.Set(key, value); err != nil {
			t.Errorf("the resource read flatten output does not fit the schema at %q: %s", key, err)
		}
	}

	// Explicitly assert the 7 new Volet B attributes are present in the
	// resource schema, so dropping one from the schema is RED even if the
	// flatten were also (wrongly) trimmed in the same change.
	for _, key := range []string{
		"ipv4_address", "ipv6_address",
		"vpc_id", "vpc_name", "private_network_id", "private_network_name",
		"static_ip_address",
	} {
		if _, declared := res.Schema[key]; !declared {
			t.Errorf("resource schema is missing the Volet B Computed attribute %q", key)
		}
		if _, emitted := flattened[key]; !emitted {
			t.Errorf("FlattenOpenIaaSNetworkAdapter does not emit the Volet B attribute %q", key)
		}
	}
}
