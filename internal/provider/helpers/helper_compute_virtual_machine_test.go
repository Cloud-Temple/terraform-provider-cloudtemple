package helpers

import (
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestConvertExtraConfigValue(t *testing.T) {
	tests := []struct {
		name        string
		key         string
		value       string
		expected    interface{}
		expectError bool
	}{
		// Tests pour les clés booléennes
		{
			name:     "boolean TRUE",
			key:      "disk.enableUUID",
			value:    "TRUE",
			expected: true,
		},
		{
			name:     "boolean FALSE",
			key:      "stealclock.enable",
			value:    "FALSE",
			expected: false,
		},
		{
			name:        "boolean invalid value",
			key:         "disk.enableUUID",
			value:       "yes",
			expectError: true,
		},
		// Tests pour les clés numériques
		{
			name:     "integer valid",
			key:      "pciPassthru.64bitMMioSizeGB",
			value:    "64",
			expected: 64,
		},
		{
			name:        "integer invalid",
			key:         "pciPassthru.64bitMMioSizeGB",
			value:       "not_a_number",
			expectError: true,
		},
		// Tests pour les clés string (non gérées)
		{
			name:     "string unmanaged key",
			key:      "svga.present",
			value:    "TRUE",
			expected: "TRUE",
		},
		{
			name:     "string ignition data",
			key:      "guestinfo.ignition.config.data",
			value:    "base64_data",
			expected: "base64_data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertExtraConfigValue(tt.key, tt.value)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("Expected %v (%T), got %v (%T)", tt.expected, tt.expected, result, result)
			}
		})
	}
}

func TestFlattenVirtualMachine_ExtraConfig(t *testing.T) {
	// Test que la fonction FlattenVirtualMachine convertit correctement les extra_config
	vm := &client.VirtualMachine{
		ID:   "test-vm-id",
		Name: "test-vm",
		ExtraConfig: []client.VirtualMachineExtraConfig{
			{
				Key:   "guestinfo.ignition.config.data",
				Value: "base64_encoded_data",
			},
			{
				Key:   "guestinfo.ignition.config.data.encoding",
				Value: "base64",
			},
			{
				Key:   "guestinfo.afterburn.initrd.network-kargs",
				Value: "ip=dhcp",
			},
			{
				Key:   "stealclock.enable",
				Value: "TRUE",
			},
			{
				Key:   "disk.enableUUID",
				Value: "TRUE",
			},
			{
				Key:   "pciPassthru.use64BitMMIO",
				Value: "TRUE",
			},
			{
				Key:   "pciPassthru.64bitMMioSizeGB",
				Value: "64",
			},
		},
		// Autres champs requis avec des valeurs par défaut
		MachineManager:    client.BaseObject{ID: "mm-id", Name: "mm-name"},
		Datacenter:        client.BaseObject{ID: "dc-id", Name: "dc-name"},
		HostCluster:       client.BaseObject{ID: "hc-id", Name: "hc-name"},
		Datastore:         client.BaseObject{ID: "ds-id", Name: "ds-name"},
		DatastoreCluster:  client.BaseObject{ID: "dsc-id", Name: "dsc-name"},
		ReplicationConfig: client.VirtualMachineReplicationConfig{},
		Storage:           client.VirtualMachineStorage{},
		BootOptions:       client.VirtualMachineBootOptions{},
	}

	result := FlattenVirtualMachine(vm)

	extraConfig, ok := result["extra_config"]
	if !ok {
		t.Fatal("extra_config not found in result")
	}

	extraConfigMap, ok := extraConfig.(map[string]string)
	if !ok {
		t.Fatalf("extra_config is not a map[string]string, got %T", extraConfig)
	}

	expected := map[string]string{
		"guestinfo.ignition.config.data":           "base64_encoded_data",
		"guestinfo.ignition.config.data.encoding":  "base64",
		"guestinfo.afterburn.initrd.network-kargs": "ip=dhcp",
		"stealclock.enable":                        "TRUE",
		"disk.enableUUID":                          "TRUE",
		"pciPassthru.use64BitMMIO":                 "TRUE",
		"pciPassthru.64bitMMioSizeGB":              "64",
	}

	if !reflect.DeepEqual(extraConfigMap, expected) {
		t.Errorf("Expected %v, got %v", expected, extraConfigMap)
	}
}

func TestFlattenVirtualMachine_EmptyExtraConfig(t *testing.T) {
	// Test avec une VM sans extra_config
	vm := &client.VirtualMachine{
		ID:                "test-vm-id",
		Name:              "test-vm",
		ExtraConfig:       []client.VirtualMachineExtraConfig{},
		MachineManager:    client.BaseObject{ID: "mm-id", Name: "mm-name"},
		Datacenter:        client.BaseObject{ID: "dc-id", Name: "dc-name"},
		HostCluster:       client.BaseObject{ID: "hc-id", Name: "hc-name"},
		Datastore:         client.BaseObject{ID: "ds-id", Name: "ds-name"},
		DatastoreCluster:  client.BaseObject{ID: "dsc-id", Name: "dsc-name"},
		ReplicationConfig: client.VirtualMachineReplicationConfig{},
		Storage:           client.VirtualMachineStorage{},
		BootOptions:       client.VirtualMachineBootOptions{},
	}

	result := FlattenVirtualMachine(vm)

	extraConfig, ok := result["extra_config"]
	if !ok {
		t.Fatal("extra_config not found in result")
	}

	extraConfigMap, ok := extraConfig.(map[string]string)
	if !ok {
		t.Fatalf("extra_config is not a map[string]string, got %T", extraConfig)
	}

	if len(extraConfigMap) != 0 {
		t.Errorf("Expected empty map, got %v", extraConfigMap)
	}
}
