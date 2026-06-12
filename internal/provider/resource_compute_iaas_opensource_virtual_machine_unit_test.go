package provider

import (
	"context"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
)

func TestAdapterNeedsUpdate(t *testing.T) {
	actual := &client.OpenIaaSNetworkAdapter{
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
		Network:        client.BaseObject{ID: "net-1"},
	}

	tests := []struct {
		name         string
		desired      map[string]interface{}
		actual       *client.OpenIaaSNetworkAdapter
		txConfigured bool
		want         bool
	}{
		{
			name:    "nil actual conservatively requests an update",
			desired: map[string]interface{}{},
			actual:  nil,
			want:    true,
		},
		{
			name: "no divergence",
			desired: map[string]interface{}{
				"network_id":      "net-1",
				"mac_address":     "AA:BB:CC:DD:EE:FF",
				"tx_checksumming": true,
			},
			actual:       actual,
			txConfigured: true,
			want:         false,
		},
		{
			name: "network divergence triggers an update",
			desired: map[string]interface{}{
				"network_id": "net-2",
			},
			actual: actual,
			want:   true,
		},
		{
			name: "mac divergence triggers an update",
			desired: map[string]interface{}{
				"mac_address": "11:22:33:44:55:66",
			},
			actual: actual,
			want:   true,
		},
		{
			name: "empty desired values never trigger an update",
			desired: map[string]interface{}{
				"network_id":  "",
				"mac_address": "",
			},
			actual: actual,
			want:   false,
		},
		{
			name: "tx divergence without explicit configuration is ignored",
			desired: map[string]interface{}{
				"tx_checksumming": false,
			},
			actual:       actual,
			txConfigured: false,
			want:         false,
		},
		{
			name: "tx divergence with explicit configuration triggers an update",
			desired: map[string]interface{}{
				"tx_checksumming": false,
			},
			actual:       actual,
			txConfigured: true,
			want:         true,
		},
		{
			name: "explicitly configured tx equal to the live value is a no-op",
			desired: map[string]interface{}{
				"tx_checksumming": true,
			},
			actual:       actual,
			txConfigured: true,
			want:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adapterNeedsUpdate(tt.desired, tt.actual, tt.txConfigured); got != tt.want {
				t.Errorf("adapterNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// txAdapterObject is the minimal raw-config shape osAdapterTxConfigured
// reads from an os_network_adapter block.
var txAdapterType = cty.Object(map[string]cty.Type{
	"tx_checksumming": cty.Bool,
})

func txRawConfig(adapters cty.Value) cty.Value {
	return cty.ObjectVal(map[string]cty.Value{
		"os_network_adapter": adapters,
	})
}

func TestOsAdapterTxConfigured(t *testing.T) {
	rawConfigType := cty.Object(map[string]cty.Type{
		"os_network_adapter": cty.List(txAdapterType),
	})
	adapters := []interface{}{
		map[string]interface{}{"id": "vif-1"},
		map[string]interface{}{"id": "vif-2"},
	}

	tests := []struct {
		name     string
		raw      cty.Value
		adapters []interface{}
		want     map[string]bool
	}{
		{
			name:     "null raw config configures nothing",
			raw:      cty.NullVal(rawConfigType),
			adapters: adapters,
			want:     map[string]bool{},
		},
		{
			name:     "unknown raw config configures nothing",
			raw:      cty.UnknownVal(rawConfigType),
			adapters: adapters,
			want:     map[string]bool{},
		},
		{
			name:     "absent os_network_adapter block configures nothing",
			raw:      txRawConfig(cty.NullVal(cty.List(txAdapterType))),
			adapters: adapters,
			want:     map[string]bool{},
		},
		{
			name:     "unknown os_network_adapter list configures nothing",
			raw:      txRawConfig(cty.UnknownVal(cty.List(txAdapterType))),
			adapters: adapters,
			want:     map[string]bool{},
		},
		{
			name: "only the block explicitly setting tx is configured",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.False}),
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.NullVal(cty.Bool)}),
			})),
			adapters: adapters,
			want:     map[string]bool{"vif-1": true},
		},
		{
			name: "fewer raw blocks than adapters leaves the extras unconfigured",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.True}),
			})),
			adapters: adapters,
			want:     map[string]bool{"vif-1": true},
		},
		{
			name: "adapter without id is skipped",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.True}),
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.True}),
			})),
			adapters: []interface{}{
				map[string]interface{}{},
				map[string]interface{}{"id": "vif-2"},
			},
			want: map[string]bool{"vif-2": true},
		},
		{
			name: "unknown explicit value counts as configured (IsNull-only semantics)",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.UnknownVal(cty.Bool)}),
			})),
			adapters: adapters[:1],
			want:     map[string]bool{"vif-1": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := osAdapterTxConfigured(tt.raw, tt.adapters)
			if len(got) != len(tt.want) {
				t.Fatalf("osAdapterTxConfigured() = %v, want %v", got, tt.want)
			}
			for id, configured := range tt.want {
				if got[id] != configured {
					t.Errorf("osAdapterTxConfigured()[%s] = %v, want %v", id, got[id], configured)
				}
			}
		})
	}
}

func TestOsNetworkAdapterUpdateSkipsUnconfiguredTx(t *testing.T) {
	actual := &client.OpenIaaSNetworkAdapter{
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
		Network:        client.BaseObject{ID: "net-1"},
	}
	adapter := map[string]interface{}{
		"id":              "vif-1",
		"network_id":      "net-1",
		"mac_address":     "AA:BB:CC:DD:EE:FF",
		"tx_checksumming": false,
	}

	// txConfigured=false: the tx divergence must not produce any PATCH. The
	// nil client guarantees a loud failure if a request were attempted.
	if diags := osNetworkAdapterUpdate(context.Background(), nil, adapter, actual, false); diags != nil {
		t.Fatalf("osNetworkAdapterUpdate() returned diagnostics for a fully converged adapter: %v", diags)
	}
}
