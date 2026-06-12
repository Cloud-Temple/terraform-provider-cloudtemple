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

func TestBuildOpenIaasVMPropertiesPatch(t *testing.T) {
	live := &client.OpenIaaSVirtualMachine{
		Name:              "vm-prod",
		CPU:               4,
		NumCoresPerSocket: 2,
		Memory:            8589934592,
		BootFirmware:      "uefi",
		HighAvailability:  "disabled",
		SecureBoot:        true,
		AutoPowerOn:       true,
	}
	intPtr := func(i int) *int { return &i }
	strPtr := func(s string) *string { return &s }
	converged := openIaasVMDesiredProperties{
		Name:              "vm-prod",
		CPU:               4,
		Memory:            8589934592,
		HighAvailability:  "disabled",
		NumCoresPerSocket: intPtr(2),
		BootFirmware:      strPtr("uefi"),
		SecureBoot:        nil,
		AutoPowerOn:       boolPtr(true),
	}

	t.Run("converged VM after create produces no PATCH", func(t *testing.T) {
		_, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, converged)
		if changed || needsReboot {
			t.Fatalf("changed=%v needsReboot=%v, want false/false", changed, needsReboot)
		}
	})

	t.Run("absent Computed values never overwrite live", func(t *testing.T) {
		desired := converged
		desired.BootFirmware = nil
		desired.NumCoresPerSocket = nil
		desired.HighAvailability = ""
		req, changed, _ := buildOpenIaasVMPropertiesPatch(live, desired)
		if changed {
			t.Fatalf("changed=true for absent values, req=%+v", req)
		}
	})

	t.Run("unconfigured cores and firmware never push even when state-stale vs live", func(t *testing.T) {
		// FF-2 finding: a state value merged through Computed could be
		// pushed when the live value changed platform-side after the
		// refresh. nil = unconfigured = never write intent.
		desired := converged
		desired.NumCoresPerSocket = nil
		desired.BootFirmware = nil
		liveDiverged := *live
		liveDiverged.NumCoresPerSocket = 4
		liveDiverged.BootFirmware = "bios"
		req, changed, _ := buildOpenIaasVMPropertiesPatch(&liveDiverged, desired)
		if changed {
			t.Fatalf("unconfigured Optional+Computed values were pushed: %+v", req)
		}
	})

	t.Run("explicitly configured firmware diverging from live is pushed", func(t *testing.T) {
		desired := converged
		desired.BootFirmware = strPtr("bios")
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if !changed || req.BootFirmware != "bios" {
			t.Fatalf("configured diverging firmware not pushed: %+v", req)
		}
		if needsReboot {
			t.Fatal("boot_firmware change must not require a reboot")
		}
	})

	t.Run("unconfigured secure_boot is never pushed even when diverging", func(t *testing.T) {
		desired := converged
		desired.SecureBoot = nil
		liveDiverged := *live
		liveDiverged.SecureBoot = false
		req, changed, _ := buildOpenIaasVMPropertiesPatch(&liveDiverged, desired)
		if changed || req.SecureBoot != nil {
			t.Fatalf("unconfigured secure_boot produced a PATCH: %+v", req)
		}
	})

	t.Run("explicit secure_boot=false diverging from live is pushed", func(t *testing.T) {
		desired := converged
		desired.SecureBoot = boolPtr(false)
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if !changed || req.SecureBoot == nil || *req.SecureBoot != false {
			t.Fatalf("explicit secure_boot=false not pushed: %+v", req)
		}
		if needsReboot {
			t.Fatal("secure_boot change must not require a reboot")
		}
	})

	t.Run("explicit secure_boot equal to live is a no-op", func(t *testing.T) {
		desired := converged
		desired.SecureBoot = boolPtr(true)
		_, changed, _ := buildOpenIaasVMPropertiesPatch(live, desired)
		if changed {
			t.Fatal("equal explicit secure_boot must not produce a PATCH")
		}
	})

	t.Run("cpu memory cores divergences require a reboot", func(t *testing.T) {
		desired := converged
		desired.CPU = 8
		desired.Memory = 17179869184
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if !changed || !needsReboot {
			t.Fatalf("changed=%v needsReboot=%v, want true/true", changed, needsReboot)
		}
		if req.CPU != 8 || req.Memory != 17179869184 {
			t.Fatalf("unexpected payload: %+v", req)
		}
		if req.Name != "" || req.BootFirmware != "" || req.SecureBoot != nil || req.AutoPowerOn != nil {
			t.Fatalf("payload carries non-diverging fields: %+v", req)
		}
	})

	t.Run("name change alone does not require a reboot", func(t *testing.T) {
		desired := converged
		desired.Name = "vm-renamed"
		req, changed, needsReboot := buildOpenIaasVMPropertiesPatch(live, desired)
		if !changed || needsReboot {
			t.Fatalf("changed=%v needsReboot=%v, want true/false", changed, needsReboot)
		}
		if req.Name != "vm-renamed" || req.CPU != 0 {
			t.Fatalf("unexpected payload: %+v", req)
		}
	})

	t.Run("auto_power_on false diverging from live true is pushed", func(t *testing.T) {
		desired := converged
		desired.AutoPowerOn = boolPtr(false)
		req, changed, _ := buildOpenIaasVMPropertiesPatch(live, desired)
		if !changed || req.AutoPowerOn == nil || *req.AutoPowerOn != false {
			t.Fatalf("diverging auto_power_on not pushed: %+v", req)
		}
	})
}

func TestBuildOpenIaasVIFPatch(t *testing.T) {
	actual := &client.OpenIaaSNetworkAdapter{
		MacAddress:     "aa:bb:cc:dd:ee:ff",
		TxChecksumming: true,
		Network:        client.BaseObject{ID: "net-1"},
	}

	t.Run("converged adapter returns nil", func(t *testing.T) {
		adapter := map[string]interface{}{
			"network_id":  "net-1",
			"mac_address": "AA:BB:CC:DD:EE:FF",
		}
		if req := buildOpenIaasVIFPatch(adapter, actual, nil); req != nil {
			t.Fatalf("converged adapter produced a PATCH: %+v", req)
		}
	})

	t.Run("mac comparison is case-insensitive", func(t *testing.T) {
		adapter := map[string]interface{}{"mac_address": "Aa:Bb:Cc:Dd:Ee:Ff"}
		if req := buildOpenIaasVIFPatch(adapter, actual, nil); req != nil {
			t.Fatalf("cosmetic mac case drift produced a PATCH: %+v", req)
		}
	})

	t.Run("only diverging fields are carried", func(t *testing.T) {
		adapter := map[string]interface{}{
			"network_id":  "net-2",
			"mac_address": "AA:BB:CC:DD:EE:FF",
		}
		req := buildOpenIaasVIFPatch(adapter, actual, nil)
		if req == nil || req.NetworkID != "net-2" || req.MAC != "" || req.TxChecksumming != nil {
			t.Fatalf("unexpected payload: %+v", req)
		}
	})

	t.Run("explicit tx false diverging is carried alone", func(t *testing.T) {
		adapter := map[string]interface{}{
			"network_id":  "net-1",
			"mac_address": "aa:bb:cc:dd:ee:ff",
		}
		req := buildOpenIaasVIFPatch(adapter, actual, boolPtr(false))
		if req == nil || req.TxChecksumming == nil || *req.TxChecksumming != false {
			t.Fatalf("explicit tx=false not carried: %+v", req)
		}
		if req.NetworkID != "" || req.MAC != "" {
			t.Fatalf("payload carries converged fields: %+v", req)
		}
	})

	t.Run("unconfigured tx diverging is not carried", func(t *testing.T) {
		adapter := map[string]interface{}{"tx_checksumming": false}
		if req := buildOpenIaasVIFPatch(adapter, actual, nil); req != nil {
			t.Fatalf("unconfigured tx produced a PATCH: %+v", req)
		}
	})
}

func TestClassifyOSDiskOnRead(t *testing.T) {
	const vmID = "vm-1"

	tests := []struct {
		name string
		disk *client.OpenIaaSVirtualDisk
		want osDiskReadAction
	}{
		{
			name: "deleted disk is dropped instead of crashing",
			disk: nil,
			want: osDiskReadDropGone,
		},
		{
			name: "read-only XO config drive on this VM is cleaned up",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "XO CloudConfigDrive",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: osDiskReadDropPlatformManaged,
		},
		{
			name: "writable user disk with the colliding name stays managed (ambiguous)",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "XO CloudConfigDrive",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false},
				},
			},
			want: osDiskReadKeep,
		},
		{
			name: "XO-named disk read-only on another VM stays managed (ambiguous)",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "XO CloudConfigDrive",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: "vm-other", ReadOnly: true},
				},
			},
			want: osDiskReadKeep,
		},
		{
			name: "XO-named disk without any VBD stays managed (ambiguous, fail-safe)",
			disk: &client.OpenIaaSVirtualDisk{Name: "XO CloudConfigDrive"},
			want: osDiskReadKeep,
		},
		{
			name: "ordinary user disk stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "data-disk",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: false, Connected: true},
				},
			},
			want: osDiskReadKeep,
		},
		{
			name: "legitimate read-only disk with another name stays managed",
			disk: &client.OpenIaaSVirtualDisk{
				Name: "shared-iso",
				VirtualMachines: []client.OpenIaaSVirtualDiskConnection{
					{ID: vmID, ReadOnly: true},
				},
			},
			want: osDiskReadKeep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := classifyOSDiskOnRead(tt.disk, vmID); got != tt.want {
				t.Errorf("classifyOSDiskOnRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDeviceConfirmedGone(t *testing.T) {
	listed := map[string]bool{"disk-1": true}
	if deviceConfirmedGone(listed, "disk-1") {
		t.Fatal("a device still present in the VM-scoped listing must NOT be confirmed gone (403 ambiguity, fail closed)")
	}
	if !deviceConfirmedGone(listed, "disk-2") {
		t.Fatal("a device absent from both the per-id read and the listing is confirmed gone")
	}
	if !deviceConfirmedGone(map[string]bool{}, "disk-1") {
		t.Fatal("an empty listing confirms the deletion")
	}
}
