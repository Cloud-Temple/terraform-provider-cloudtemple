package helpers

import (
	"sort"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// primaryAddress projects the device-0 primary address of a given family
// (ipv4 / ipv6) out of the OpenIaaS VM addresses map. The API keys it by a
// composite "<device>/<family>/<index>" (e.g. "0/ipv4/0"), so a flat struct
// could never bind it (#238). The exact device-0 key wins; if it is absent we
// fall back to the FIRST "*/<family>/*" key after sorting the keys lexically,
// so the result is DETERMINISTIC and never depends on Go's randomized map
// iteration. A nil/empty map yields "" without panicking.
func primaryAddress(addresses map[string]string, family string) string {
	if len(addresses) == 0 {
		return ""
	}
	if v, ok := addresses["0/"+family+"/0"]; ok {
		return v
	}
	infix := "/" + family + "/"
	keys := make([]string, 0, len(addresses))
	for k := range addresses {
		if strings.Contains(k, infix) {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return ""
	}
	sort.Strings(keys)
	return addresses[keys[0]]
}

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

	// Mapper les addresses. The API returns a composite-keyed object
	// ("0/ipv4/0", "0/ipv6/0", ...); the existing {ipv4, ipv6} state block
	// stays unchanged and is populated from the device-0 primary entries.
	// Addresses beyond device 0 are intentionally deferred (a richer multi-NIC
	// shape would be a state-breaking schema change, #238).
	addresses := []map[string]interface{}{
		{
			"ipv4": primaryAddress(vm.Addresses, "ipv4"),
			"ipv6": primaryAddress(vm.Addresses, "ipv6"),
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

// cloudConfigDriveName is the name XO gives to the cloud-init config drive
// it attaches at deploy time.
const cloudConfigDriveName = "XO CloudConfigDrive"

// IsPlatformManagedDisk reports whether the disk is managed by the platform
// (cloud-init config drive, attached read-only by XO at deploy time) and must
// never be reconciled as an os_disk: capturing it — which is timing dependent
// at create — produces a permanent removal drift in every subsequent plan.
// Both discriminators are required: the exact XO naming alone could hide a
// legitimate user disk carrying the same name, and a read-only criterion
// alone could drop legitimate read-only disks without a formal API
// invariant. Only a disk attached read-only to this VM under the exact XO
// platform naming is excluded; when the VBD is absent Find returns a zero
// value, so the default stays fail-safe (the disk remains managed).
func IsPlatformManagedDisk(osDisk *client.OpenIaaSVirtualDisk, virtualMachineId string) bool {
	if osDisk == nil {
		// A nil disk (the API maps a deleted/forbidden disk to nil) is not a
		// platform-managed disk; never dereference it (#320).
		return false
	}
	if osDisk.Name != cloudConfigDriveName {
		return false
	}
	vbd := Find(osDisk.VirtualMachines, func(virtualMachine client.OpenIaaSVirtualDiskConnection) bool {
		return virtualMachine.ID == virtualMachineId
	})
	return vbd.ReadOnly
}

func FlattenOpenIaaSOSDisksData(osDisks []*client.OpenIaaSVirtualDisk, virtualMachineId string) []interface{} {
	disks := make([]interface{}, 0, len(osDisks))

	for _, osDisk := range osDisks {
		// Skip a nil disk (the API maps a deleted/forbidden disk to nil): it must
		// never be dereferenced nor appended as a nil entry (#320). append-based
		// (not index-based), so skipping does not shift the kept disks.
		if osDisk == nil {
			continue
		}
		if IsPlatformManagedDisk(osDisk, virtualMachineId) {
			continue
		}
		disks = append(disks, FlattenOpenIaaSOSDiskData(osDisk, virtualMachineId))
	}

	return disks
}

func FlattenOpenIaaSOSDiskData(osDisk *client.OpenIaaSVirtualDisk, virtualMachineId string) interface{} {
	if osDisk == nil {
		// The API maps a deleted/forbidden disk to a nil read (403 -> nil).
		// Refreshing a VM whose OS disk disappeared out-of-band must not panic
		// here (#320); callers skip a nil entry. Defense-in-depth on top of
		// classifyOSDiskOnRead, which already drops a nil disk on the read path.
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
