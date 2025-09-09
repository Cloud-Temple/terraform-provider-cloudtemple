package helpers

import (
	"context"
	"fmt"
	"strconv"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// FlattenVirtualMachine convertit un objet VirtualMachine en une map compatible avec le schéma Terraform
func FlattenVirtualMachine(vm *client.VirtualMachine) map[string]interface{} {
	// Aplatir les alarmes déclenchées
	triggeredAlarms := make([]map[string]interface{}, len(vm.TriggeredAlarms))
	for i, alarm := range vm.TriggeredAlarms {
		triggeredAlarms[i] = map[string]interface{}{
			"id":     alarm.ID,
			"status": alarm.Status,
		}
	}

	// Aplatir la configuration de réplication
	replicationConfig := []map[string]interface{}{}
	if vm.ReplicationConfig.VmReplicationId != "" {
		disks := make([]map[string]interface{}, len(vm.ReplicationConfig.Disk))
		for i, disk := range vm.ReplicationConfig.Disk {
			disks[i] = map[string]interface{}{
				"key":                 disk.Key,
				"disk_replication_id": disk.DiskReplicationId,
			}
		}

		replicationConfig = append(replicationConfig, map[string]interface{}{
			"generation":              vm.ReplicationConfig.Generation,
			"vm_replication_id":       vm.ReplicationConfig.VmReplicationId,
			"rpo":                     vm.ReplicationConfig.Rpo,
			"quiesce_guest_enabled":   vm.ReplicationConfig.QuiesceGuestEnabled,
			"paused":                  vm.ReplicationConfig.Paused,
			"opp_updates_enabled":     vm.ReplicationConfig.OppUpdatesEnabled,
			"net_compression_enabled": vm.ReplicationConfig.NetCompressionEnabled,
			"net_encryption_enabled":  vm.ReplicationConfig.NetEncryptionEnabled,
			"encryption_destination":  vm.ReplicationConfig.EncryptionDestination,
			"disk":                    disks,
		})
	}

	// Aplatir la configuration supplémentaire
	extraConfigMap := make(map[string]string)
	for _, config := range vm.ExtraConfig {
		extraConfigMap[config.Key] = config.Value
	}

	// Aplatir le stockage
	storage := []map[string]interface{}{}
	if vm.Storage.Committed > 0 || vm.Storage.Uncommitted > 0 {
		storage = append(storage, map[string]interface{}{
			"committed":   vm.Storage.Committed,
			"uncommitted": vm.Storage.Uncommitted,
		})
	}

	// Aplatir les options de démarrage
	bootOptions := []map[string]interface{}{}
	if vm.BootOptions.Firmware != "" {
		bootOptions = append(bootOptions, map[string]interface{}{
			"firmware":                vm.BootOptions.Firmware,
			"boot_delay":              vm.BootOptions.BootDelay,
			"enter_bios_setup":        vm.BootOptions.EnterBIOSSetup,
			"boot_retry_enabled":      vm.BootOptions.BootRetryEnabled,
			"boot_retry_delay":        vm.BootOptions.BootRetryDelay,
			"efi_secure_boot_enabled": vm.BootOptions.EFISecureBootEnabled,
		})
	}

	return map[string]interface{}{
		"name":                               vm.Name,
		"moref":                              vm.Moref,
		"machine_manager_id":                 vm.MachineManager.ID,
		"machine_manager_name":               vm.MachineManager.Name,
		"datacenter_id":                      vm.Datacenter.ID,
		"host_cluster_id":                    vm.HostCluster.ID,
		"datastore_id":                       vm.Datastore.ID,
		"datastore_name":                     vm.Datastore.Name,
		"datastore_cluster_id":               vm.DatastoreCluster.ID,
		"consolidation_needed":               vm.ConsolidationNeeded,
		"template":                           vm.Template,
		"power_state":                        vm.PowerState,
		"hardware_version":                   vm.HardwareVersion,
		"num_cores_per_socket":               vm.NumCoresPerSocket,
		"operating_system_name":              vm.OperatingSystemName,
		"guest_operating_system_moref":       vm.OperatingSystemMoref,
		"cpu":                                vm.Cpu,
		"cpu_hot_add_enabled":                vm.CpuHotAddEnabled,
		"cpu_hot_remove_enabled":             vm.CpuHotRemoveEnabled,
		"memory_hot_add_enabled":             vm.MemoryHotAddEnabled,
		"memory":                             vm.Memory,
		"cpu_usage":                          vm.CpuUsage,
		"memory_usage":                       vm.MemoryUsage,
		"tools":                              vm.Tools,
		"tools_version":                      vm.ToolsVersion,
		"distributed_virtual_port_group_ids": vm.DistributedVirtualPortGroupIds,
		"spp_mode":                           vm.SppMode,
		"snapshoted":                         vm.Snapshoted,
		"triggered_alarms":                   triggeredAlarms,
		"replication_config":                 replicationConfig,
		"extra_config":                       extraConfigMap,
		"storage":                            storage,
		"boot_options":                       bootOptions,
		"expose_hardware_virtualization":     vm.ExposeHardwareVirtualization,
	}
}

func BuildGuestOSCustomizationRequest(ctx context.Context, d *schema.ResourceData) *client.CustomizeGuestOSRequest {
	dnsServerList := []string{}
	for _, policy := range d.Get("customize.0.network_config.0.dns_server_list").(*schema.Set).List() {
		dnsServerList = append(dnsServerList, policy.(string))
	}

	dnsSuffixList := []string{}
	for _, policy := range d.Get("customize.0.network_config.0.dns_suffix_list").(*schema.Set).List() {
		dnsSuffixList = append(dnsSuffixList, policy.(string))
	}

	adaptersConfig := []*client.CustomAdapterConfig{}
	for _, adapter := range d.Get("customize.0.network_config.0.adapters").([]interface{}) {
		adaptersConfig = append(adaptersConfig, &client.CustomAdapterConfig{
			MacAddress: adapter.(map[string]interface{})["mac_address"].(string),
			IpAddress:  adapter.(map[string]interface{})["ip_address"].(string),
			SubnetMask: adapter.(map[string]interface{})["subnet_mask"].(string),
			Gateway:    adapter.(map[string]interface{})["gateway"].(string),
		})
	}

	customizationRequest := &client.CustomizeGuestOSRequest{
		NetworkConfig: &client.CustomGuestNetworkConfig{
			Hostname:      d.Get("customize.0.network_config.0.hostname").(string),
			Domain:        d.Get("customize.0.network_config.0.domain").(string),
			DnsServerList: dnsServerList,
			DnsSuffixList: dnsSuffixList,
			Adapters:      adaptersConfig,
		},
	}

	if len(d.Get("customize.0.windows_config").([]interface{})) > 0 {
		customizationRequest.WindowsConfig = &client.CustomGuestWindowsConfig{
			AutoLogon:           d.Get("customize.0.windows_config.0.auto_logon").(bool),
			AutoLogonCount:      d.Get("customize.0.windows_config.0.auto_logon_count").(int),
			TimeZone:            d.Get("customize.0.windows_config.0.timezone").(int),
			Password:            d.Get("customize.0.windows_config.0.password").(string),
			JoinDomain:          d.Get("customize.0.windows_config.0.domain.0.name").(string),
			DomainAdmin:         d.Get("customize.0.windows_config.0.domain.0.admin_username").(string),
			DomainAdminPassword: d.Get("customize.0.windows_config.0.domain.0.admin_password").(string),
		}
	}

	return customizationRequest
}

func GetPowerRecommendation(vm *client.VirtualMachine, powerState string, ctx context.Context, c *client.Client) (*client.VirtualMachinePowerRecommendation, error) {
	var err error
	var recommendations []*client.VirtualMachinePowerRecommendation

	if powerState == "on" || powerState == "running" {
		recommendations, err = c.Compute().VirtualMachine().Recommendation(ctx, &client.VirtualMachineRecommendationFilter{
			Id:            vm.ID,
			DatacenterId:  vm.Datacenter.ID,
			HostClusterId: vm.HostCluster.ID,
		})
		if err != nil {
			return nil, err
		}
	}

	var recommendation *client.VirtualMachinePowerRecommendation
	if len(recommendations) > 0 {
		recommendation = &client.VirtualMachinePowerRecommendation{
			Key:           recommendations[0].Key,
			HostClusterId: recommendations[0].HostClusterId,
		}
	} else {
		recommendation = nil
	}

	return recommendation, err
}

func UpdateNestedMapItems(d *schema.ResourceData, nestedMapItems []interface{}, key string) []interface{} {
	nestedMaps := make([]interface{}, len(nestedMapItems))

	for i, mapItems := range nestedMapItems {
		nestedMaps[i] = UpdateMapItems(d, mapItems, key, i)
	}

	return nestedMaps
}

func UpdateMapItems(d *schema.ResourceData, mapItems interface{}, key string, index int) interface{} {
	m := mapItems.(map[string]interface{})

	if value, ok := d.GetOk(fmt.Sprintf("%s.%d", key, index)); ok {

		for k, v := range value.(map[string]interface{}) {
			if _, ok := d.GetOk(fmt.Sprintf("%s.%d.%s", key, index, k)); ok {
				m[k] = v
			}
		}
	}
	return m
}

func FlattenOSDisksData(osDisks []*client.VirtualDisk) []interface{} {
	if osDisks != nil {
		disks := make([]interface{}, len(osDisks))

		for i, osDisk := range osDisks {
			disks[i] = FlattenOSDiskData(osDisk)
		}

		return disks
	}

	return make([]interface{}, 0)
}

func FlattenOSDiskData(osDisk *client.VirtualDisk) interface{} {
	disk := make(map[string]interface{})

	disk["id"] = osDisk.ID
	disk["machine_manager_id"] = osDisk.MachineManager.ID
	disk["name"] = osDisk.Name
	disk["capacity"] = osDisk.Capacity
	disk["disk_unit_number"] = osDisk.DiskUnitNumber
	disk["controller_bus_number"] = osDisk.Controller.BusNumber
	disk["datastore_id"] = osDisk.Datastore.ID
	disk["datastore_name"] = osDisk.Datastore.Name
	disk["instant_access"] = osDisk.InstantAccess
	disk["native_id"] = osDisk.NativeId
	disk["disk_path"] = osDisk.DiskPath
	disk["provisioning_type"] = osDisk.ProvisioningType
	disk["disk_mode"] = osDisk.DiskMode
	disk["editable"] = osDisk.Editable

	return disk
}

func FlattenOSNetworkAdaptersData(osNetworkAdapters []*client.NetworkAdapter) []interface{} {
	if osNetworkAdapters != nil {
		networkAdapters := make([]interface{}, len(osNetworkAdapters))

		for i, osNetworkAdapter := range osNetworkAdapters {
			networkAdapters[i] = FlattenOSNetworkAdapterData(osNetworkAdapter)
		}

		return networkAdapters
	}

	return make([]interface{}, 0)
}

func FlattenOSNetworkAdapterData(osNetworkAdapter *client.NetworkAdapter) interface{} {
	networkAdapter := make(map[string]interface{})

	networkAdapter["id"] = osNetworkAdapter.ID
	networkAdapter["name"] = osNetworkAdapter.Name
	networkAdapter["network_id"] = osNetworkAdapter.NetworkId
	networkAdapter["type"] = osNetworkAdapter.Type
	networkAdapter["mac_type"] = osNetworkAdapter.MacType
	networkAdapter["mac_address"] = osNetworkAdapter.MacAddress
	networkAdapter["connected"] = osNetworkAdapter.Connected
	networkAdapter["auto_connect"] = osNetworkAdapter.AutoConnect

	return networkAdapter
}

// convertExtraConfigValue convertit une valeur string vers le type approprié selon la clé
func ConvertExtraConfigValue(key, value string) (interface{}, error) {
	switch key {
	// Clés booléennes VMware (strictes)
	case "disk.enableUUID", "stealclock.enable", "pciPassthru.use64BitMMIO":
		switch value {
		case "TRUE", "true":
			return true, nil
		case "FALSE", "false":
			return false, nil
		default:
			return nil, fmt.Errorf("invalid boolean value '%s' for key '%s', expected 'TRUE' or 'FALSE'", value, key)
		}

	// Clés numériques
	case "pciPassthru.64bitMMioSizeGB":
		if i, err := strconv.Atoi(value); err != nil {
			return nil, fmt.Errorf("invalid integer value '%s' for key '%s': %v", value, key, err)
		} else {
			return i, nil
		}

	// Clés string (par défaut - toutes les non gérées)
	default:
		return value, nil
	}
}
