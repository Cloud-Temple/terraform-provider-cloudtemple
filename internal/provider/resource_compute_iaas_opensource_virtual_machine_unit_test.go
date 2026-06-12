package provider

import (
	"context"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/go-cty/cty"
)

func boolPtr(b bool) *bool { return &b }

func TestAdapterNeedsUpdate(t *testing.T) {
	actual := &client.OpenIaaSNetworkAdapter{
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
		Network:        client.BaseObject{ID: "net-1"},
	}

	tests := []struct {
		name    string
		desired map[string]interface{}
		actual  *client.OpenIaaSNetworkAdapter
		txWant  *bool
		want    bool
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
				"network_id":  "net-1",
				"mac_address": "AA:BB:CC:DD:EE:FF",
			},
			actual: actual,
			txWant: boolPtr(true),
			want:   false,
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
			name: "tx divergence in the merged map without explicit config is ignored",
			desired: map[string]interface{}{
				"tx_checksumming": false,
			},
			actual: actual,
			txWant: nil,
			want:   false,
		},
		{
			name: "explicit raw false diverging from the live value triggers an update",
			desired: map[string]interface{}{
				// First apply: the merged map is seeded with the live API
				// value, swallowing the explicit false from the config.
				"tx_checksumming": true,
			},
			actual: actual,
			txWant: boolPtr(false),
			want:   true,
		},
		{
			name: "explicit raw value equal to the live value is a no-op",
			desired: map[string]interface{}{
				"tx_checksumming": true,
			},
			actual: actual,
			txWant: boolPtr(true),
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := adapterNeedsUpdate(tt.desired, tt.actual, tt.txWant); got != tt.want {
				t.Errorf("adapterNeedsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

// txAdapterType is the minimal raw-config shape osAdapterTxConfigured reads
// from an os_network_adapter block.
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
		want     map[string]*bool
	}{
		{
			name:     "null raw config configures nothing",
			raw:      cty.NullVal(rawConfigType),
			adapters: adapters,
			want:     map[string]*bool{},
		},
		{
			name:     "unknown raw config configures nothing",
			raw:      cty.UnknownVal(rawConfigType),
			adapters: adapters,
			want:     map[string]*bool{},
		},
		{
			name:     "absent os_network_adapter block configures nothing",
			raw:      txRawConfig(cty.NullVal(cty.List(txAdapterType))),
			adapters: adapters,
			want:     map[string]*bool{},
		},
		{
			name:     "unknown os_network_adapter list configures nothing",
			raw:      txRawConfig(cty.UnknownVal(cty.List(txAdapterType))),
			adapters: adapters,
			want:     map[string]*bool{},
		},
		{
			name: "only the block explicitly setting tx carries its raw value",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.False}),
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.NullVal(cty.Bool)}),
			})),
			adapters: adapters,
			want:     map[string]*bool{"vif-1": boolPtr(false)},
		},
		{
			name: "fewer raw blocks than adapters leaves the extras unconfigured",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.True}),
			})),
			adapters: adapters,
			want:     map[string]*bool{"vif-1": boolPtr(true)},
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
			want: map[string]*bool{"vif-2": boolPtr(true)},
		},
		{
			name: "unknown explicit value stays unconfigured (no concrete value to push)",
			raw: txRawConfig(cty.ListVal([]cty.Value{
				cty.ObjectVal(map[string]cty.Value{"tx_checksumming": cty.UnknownVal(cty.Bool)}),
			})),
			adapters: adapters[:1],
			want:     map[string]*bool{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := osAdapterTxConfigured(tt.raw, tt.adapters)
			if len(got) != len(tt.want) {
				t.Fatalf("osAdapterTxConfigured() = %v, want %v", got, tt.want)
			}
			for id, wantVal := range tt.want {
				gotVal, ok := got[id]
				if !ok || gotVal == nil {
					t.Errorf("osAdapterTxConfigured()[%s] missing, want %v", id, *wantVal)
					continue
				}
				if *gotVal != *wantVal {
					t.Errorf("osAdapterTxConfigured()[%s] = %v, want %v", id, *gotVal, *wantVal)
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

	// txWant=nil: the merged-map tx divergence must not produce any PATCH.
	// The nil client guarantees a loud failure if a request were attempted.
	if diags := osNetworkAdapterUpdate(context.Background(), nil, adapter, actual, nil); diags != nil {
		t.Fatalf("osNetworkAdapterUpdate() returned diagnostics for a fully converged adapter: %v", diags)
	}

	// txWant equal to the live value: no divergence, no PATCH either.
	if diags := osNetworkAdapterUpdate(context.Background(), nil, adapter, actual, boolPtr(true)); diags != nil {
		t.Fatalf("osNetworkAdapterUpdate() returned diagnostics for an equal explicit value: %v", diags)
	}
}
