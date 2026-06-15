package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenOpenIaaSVirtualMachine convertit un objet OpenIaaSVirtualMachine en une map compatible avec le schéma Terraform
func FlattenOpenIaaSVirtualMachine(vm *client.OpenIaaSVirtualMachine) map[string]interface{} {
	// Mapper le dvd_drive
	dvdDrive := []map[string]interface{}{
		{
			"name":     vm.DvdDrive.Name,
			"attached": vm.DvdDrive.Attached,
		},
	}

	// Mapper les tools (Deprecated, use pvDrivers and managementAgent instead)
	tools := []map[string]interface{}{
		{
			"detected": vm.PVDrivers.Detected,
			"version":  vm.PVDrivers.Version,
		},
	}

	pvDrivers := []map[string]interface{}{
		{
			"detected":       vm.PVDrivers.Detected,
			"version":        vm.PVDrivers.Version,
			"are_up_to_date": vm.PVDrivers.AreUpToDate,
		},
	}

	managementAgent := []map[string]interface{}{
		{
			"detected": vm.ManagementAgent.Detected,
		},
	}

	// Mapper les addresses
	addresses := []map[string]interface{}{
		{
			"ipv4": vm.Addresses.IPv4,
			"ipv6": vm.Addresses.IPv6,
		},
	}

	return map[string]interface{}{
		"name":                  vm.Name,
		"internal_id":           vm.InternalID,
		"power_state":           vm.PowerState,
		"secure_boot":           vm.SecureBoot,
		"boot_firmware":         vm.BootFirmware,
		"auto_power_on":         vm.AutoPowerOn,
		"high_availability":     vm.HighAvailability,
		"dvd_drive":             dvdDrive,
		"boot_order":            vm.BootOrder,
		"operating_system_name": vm.OperatingSystemName,
		"cpu":                   vm.CPU,
		"num_cores_per_socket":  vm.NumCoresPerSocket,
		"memory":                vm.Memory,
		"tools":                 tools,
		"pv_drivers":            pvDrivers,
		"management_agent":      managementAgent,
		"addresses":             addresses,
		"host_id":               vm.Host.ID,
		"pool_id":               vm.Pool.ID,
		"machine_manager_id":    vm.MachineManager.ID,
	}
}

func FlattenOpenIaaSOSDisksData(osDisks []*client.OpenIaaSVirtualDisk, virtualMachineId string) []interface{} {
	disks := make([]interface{}, 0, len(osDisks))
	for _, osDisk := range osDisks {
		// Skip a nil disk (the API maps a deleted/forbidden disk to nil): it
		// must never be dereferenced nor appended as a nil entry (#320).
		if osDisk == nil {
			continue
		}
		disks = append(disks, FlattenOpenIaaSOSDiskData(osDisk, virtualMachineId))
	}
	return disks
}

func FlattenOpenIaaSOSDiskData(osDisk *client.OpenIaaSVirtualDisk, virtualMachineId string) interface{} {
	if osDisk == nil {
		// The API maps a deleted/forbidden disk to a nil read (403 -> nil).
		// Dereferencing it while refreshing a VM whose OS disk disappeared
		// out-of-band panics (#320). Return nil; callers skip a nil entry.
		return nil
	}
	vbd := Find(osDisk.VirtualMachines, func(virtualMachine client.OpenIaaSVirtualDiskConnection) bool {
		return virtualMachine.ID == virtualMachineId
	})

	return map[string]interface{}{
		"id":                    osDisk.ID,
		"name":                  osDisk.Name,
		"size":                  osDisk.Size,
		"description":           osDisk.Description,
		"storage_repository_id": osDisk.StorageRepository.ID,
		"connected":             vbd.Connected,
	}
}

func FlattenOpenIaaSOSNetworkAdaptersData(osNetworkAdapters []*client.OpenIaaSNetworkAdapter) []interface{} {
	if osNetworkAdapters != nil {
		networkAdapters := make([]interface{}, len(osNetworkAdapters))

		for i, osNetworkAdapter := range osNetworkAdapters {
			networkAdapters[i] = FlattenOpenIaaSOSNetworkAdapterData(osNetworkAdapter)
		}

		return networkAdapters
	}

	return make([]interface{}, 0)
}

func FlattenOpenIaaSOSNetworkAdapterData(osNetworkAdapter *client.OpenIaaSNetworkAdapter) interface{} {
	return map[string]interface{}{
		"id":              osNetworkAdapter.ID,
		"name":            osNetworkAdapter.Name,
		"mac_address":     osNetworkAdapter.MacAddress,
		"mtu":             osNetworkAdapter.MTU,
		"attached":        osNetworkAdapter.Attached,
		"tx_checksumming": osNetworkAdapter.TxChecksumming,
		"network_id":      osNetworkAdapter.Network.ID,
	}
}
