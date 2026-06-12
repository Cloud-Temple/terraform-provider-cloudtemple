package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
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
