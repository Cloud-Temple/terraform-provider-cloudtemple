package helpers

import (
	"reflect"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

func TestExpandExtraConfig(t *testing.T) {
	tests := []struct {
		name     string
		input    map[string]interface{}
		expected []client.VirtualMachineExtraConfig
	}{
		{
			name:     "nil input",
			input:    nil,
			expected: nil,
		},
		{
			name:     "empty map",
			input:    map[string]interface{}{},
			expected: []client.VirtualMachineExtraConfig{},
		},
		{
			name: "single config",
			input: map[string]interface{}{
				"stealclock.enable": "TRUE",
			},
			expected: []client.VirtualMachineExtraConfig{
				{
					Key:   "stealclock.enable",
					Value: "TRUE",
				},
			},
		},
		{
			name: "multiple configs",
			input: map[string]interface{}{
				"guestinfo.ignition.config.data":           "base64_encoded_data",
				"guestinfo.ignition.config.data.encoding":  "base64",
				"guestinfo.afterburn.initrd.network-kargs": "ip=dhcp",
				"stealclock.enable":                        "TRUE",
				"disk.enableUUID":                          "TRUE",
				"pciPassthru.use64BitMMIO":                 "TRUE",
				"pciPassthru.64bitMMioSizeGB":              "64",
			},
			expected: []client.VirtualMachineExtraConfig{
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
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExpandExtraConfig(tt.input)

			if tt.expected == nil {
				if result != nil {
					t.Errorf("Expected nil, got %v", result)
				}
				return
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d items, got %d", len(tt.expected), len(result))
				return
			}

			// Convert to maps for easier comparison since order might vary
			resultMap := make(map[string]string)
			for _, item := range result {
				resultMap[item.Key] = item.Value
			}

			expectedMap := make(map[string]string)
			for _, item := range tt.expected {
				expectedMap[item.Key] = item.Value
			}

			if !reflect.DeepEqual(resultMap, expectedMap) {
				t.Errorf("Expected %v, got %v", expectedMap, resultMap)
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
