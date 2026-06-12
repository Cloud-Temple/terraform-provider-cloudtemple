package provider

import (
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

// datasourceFlattenCheck registers a datasource whose flatten output must
// fit its declared schema. When a flatten helper emits a key the schema
// does not declare (or with an incompatible shape), d.Set fails at runtime
// with "Invalid address to set" and the datasource becomes unusable — the
// #243 class. Adding an entry here is a one-liner: coverage must grow with
// every new datasource.
type datasourceFlattenCheck struct {
	name       string
	datasource *schema.Resource
	// rootKey is the computed list attribute for list datasources; empty
	// for single-object datasources whose Read sets the flatten map per key.
	rootKey   string
	flattened map[string]interface{}
}

// assertFlattenFitsSchema validates the flatten output against the schema
// through the real schema writer, reporting the exact offending address.
func assertFlattenFitsSchema(t *testing.T, check datasourceFlattenCheck) {
	t.Helper()
	d := schema.TestResourceDataRaw(t, check.datasource.Schema, map[string]interface{}{})

	if check.rootKey != "" {
		if _, declared := check.datasource.Schema[check.rootKey]; !declared {
			t.Errorf("%s: root key %q is not declared in the schema", check.name, check.rootKey)
			return
		}
		if err := d.Set(check.rootKey, []interface{}{check.flattened}); err != nil {
			t.Errorf("%s: the flatten output does not fit the %q schema: %s", check.name, check.rootKey, err)
		}
		return
	}

	for key, value := range check.flattened {
		if _, declared := check.datasource.Schema[key]; !declared {
			t.Errorf("%s: the flatten helper emits %q which the schema does not declare", check.name, key)
			continue
		}
		if err := d.Set(key, value); err != nil {
			t.Errorf("%s: the flatten output does not fit the schema at %q: %s", check.name, key, err)
		}
	}
}

// Fully-populated client fixtures: every field carries a non-zero value so
// that type and shape mismatches cannot hide behind zero values.

func fixtureOpenIaasPool() *client.OpenIaasPool {
	pool := &client.OpenIaasPool{
		ID:                      "pool-1",
		MachineManager:          client.BaseObject{ID: "mm-1", Name: "xoa"},
		InternalID:              "internal-1",
		Name:                    "pool-name",
		Label:                   "pool-label",
		HighAvailabilityEnabled: true,
		Master:                  "host-master",
		Hosts:                   []string{"host-1", "host-2"},
	}
	pool.Memory.Usage = 10
	pool.Memory.Size = 100
	pool.Cpu.Cores = 8
	pool.Cpu.Sockets = 2
	pool.Type.Key = "xcp-ng"
	pool.Type.Description = "XCP-ng pool"
	return pool
}

func fixtureOpenIaasHost() *client.OpenIaaSHost {
	host := &client.OpenIaaSHost{
		ID:              "host-1",
		MachineManager:  client.BaseObject{ID: "mm-1", Name: "xoa"},
		InternalId:      "internal-1",
		Name:            "host-name",
		Master:          true,
		Uptime:          3600,
		PowerState:      "Running",
		RebootRequired:  true,
		VirtualMachines: []string{"vm-1"},
	}
	host.Pool.ID = "pool-1"
	host.Pool.Name = "pool-name"
	host.Pool.Type.Key = "xcp-ng"
	host.Pool.Type.Description = "XCP-ng pool"
	host.UpdateData.MaintenanceMode = true
	host.UpdateData.Status = "ok"
	host.Metrics.XOA.Version = "5.0"
	host.Metrics.XOA.FullName = "XOA 5.0"
	host.Metrics.XOA.Build = "build-1"
	host.Metrics.Memory.Usage = 10
	host.Metrics.Memory.Size = 100
	host.Metrics.Cpu.Sockets = 2
	host.Metrics.Cpu.Cores = 8
	host.Metrics.Cpu.Model = "EPYC"
	host.Metrics.Cpu.ModelName = "AMD EPYC"
	return host
}

func fixtureOpenIaasNetwork() *client.OpenIaaSNetwork {
	return &client.OpenIaaSNetwork{
		ID:                         "net-1",
		MachineManager:             client.BaseObject{ID: "mm-1", Name: "xoa"},
		InternalID:                 "internal-1",
		Name:                       "net-name",
		Pool:                       client.BaseObject{ID: "pool-1", Name: "pool-name"},
		MaximumTransmissionUnit:    1500,
		NetworkAdapters:            []string{"vif-1"},
		NetworkBlockDevice:         true,
		InsecureNetworkBlockDevice: true,
	}
}

func fixtureOpenIaasNetworkAdapter() *client.OpenIaaSNetworkAdapter {
	return &client.OpenIaaSNetworkAdapter{
		ID:               "vif-1",
		Name:             "vif-name",
		InternalID:       "internal-1",
		VirtualMachineID: "vm-1",
		MacAddress:       "aa:bb:cc:dd:ee:ff",
		MTU:              1500,
		Attached:         true,
		TxChecksumming:   true,
		Network:          client.BaseObject{ID: "net-1", Name: "net-name"},
		MachineManager:   client.BaseObject{ID: "mm-1", Name: "xoa"},
	}
}

func fixtureOpenIaasSnapshot() *client.OpenIaaSSnapshot {
	return &client.OpenIaaSSnapshot{
		ID:               "snap-1",
		Description:      "snapshot description",
		VirtualMachineID: "vm-1",
		Name:             "snap-name",
		CreateTime:       1700000000,
	}
}

func fixtureOpenIaasReplicationPolicy() *client.OpenIaaSReplicationPolicy {
	policy := &client.OpenIaaSReplicationPolicy{
		ID:   "rp-1",
		Name: "rp-name",
	}
	policy.StorageRepository.ID = "sr-1"
	policy.StorageRepository.Name = "sr-name"
	policy.Pool.ID = "pool-1"
	policy.Pool.Name = "pool-name"
	policy.Pool.Label = "pool-label"
	policy.MachineManager.ID = "mm-1"
	policy.MachineManager.Name = "xoa"
	policy.LastRun.Start = 1700000000
	policy.LastRun.End = 1700000100
	policy.LastRun.Status = "success"
	policy.Interval.Hours = 1
	policy.Interval.Minutes = 30
	return policy
}

func fixtureOpenIaasStorageRepository() *client.OpenIaaSStorageRepository {
	return &client.OpenIaaSStorageRepository{
		ID:              "sr-1",
		InternalId:      "internal-1",
		Name:            "sr-name",
		Description:     "sr description",
		MaintenanceMode: true,
		MaxCapacity:     1000,
		FreeCapacity:    500,
		StorageType:     "lvm",
		VirtualDisks:    []string{"disk-1"},
		Shared:          true,
		Accessible:      1,
		Host:            client.BaseObject{ID: "host-1", Name: "host-name"},
		Pool:            client.BaseObject{ID: "pool-1", Name: "pool-name"},
		MachineManager:  client.BaseObject{ID: "mm-1", Name: "xoa"},
	}
}

func fixtureOpenIaasTemplate() *client.OpenIaasTemplate {
	return &client.OpenIaasTemplate{
		ID:                "tpl-1",
		MachineManager:    client.BaseObject{ID: "mm-1", Name: "xoa"},
		InternalID:        "internal-1",
		Name:              "tpl-name",
		CPU:               4,
		NumCoresPerSocket: 2,
		Memory:            8589934592,
		PowerState:        "Halted",
		Snapshots:         []string{"snap-1"},
		SLAPolicies:       []string{"sla-1"},
		Disks: []client.TemplateDisk{{
			ID:                "disk-1",
			Name:              "disk-name",
			Description:       "disk description",
			Size:              1024,
			StorageRepository: client.BaseObject{ID: "sr-1", Name: "sr-name"},
		}},
		NetworkAdapters: []client.TemplateNetworkAdapter{{
			Name:       "vif-name",
			MacAddress: "aa:bb:cc:dd:ee:ff",
			MTU:        1500,
			Attached:   true,
			Network:    client.BaseObject{ID: "net-1", Name: "net-name"},
		}},
	}
}

func fixtureOpenIaasVirtualDisk() *client.OpenIaaSVirtualDisk {
	disk := &client.OpenIaaSVirtualDisk{
		ID:          "disk-1",
		InternalID:  "internal-1",
		Name:        "disk-name",
		Description: "disk description",
		Size:        1024,
		Usage:       512,
		IsSnapshot:  true,
		StorageRepository: client.BaseObject{
			ID:   "sr-1",
			Name: "sr-name",
		},
		VirtualMachines: []client.OpenIaaSVirtualDiskConnection{{
			ID:        "vm-1",
			Name:      "vm-name",
			ReadOnly:  true,
			Connected: true,
		}},
	}
	disk.Templates = append(disk.Templates, struct {
		ID       string
		Name     string
		ReadOnly bool
	}{ID: "tpl-1", Name: "tpl-name", ReadOnly: true})
	return disk
}

func fixtureOpenIaasVirtualMachine() *client.OpenIaaSVirtualMachine {
	vm := &client.OpenIaaSVirtualMachine{
		ID:                  "vm-1",
		Name:                "vm-name",
		InternalID:          "internal-1",
		PowerState:          "Running",
		SecureBoot:          true,
		HighAvailability:    "restart",
		BootFirmware:        "uefi",
		AutoPowerOn:         true,
		BootOrder:           []string{"Hard-Drive"},
		OperatingSystemName: "Ubuntu",
		CPU:                 4,
		NumCoresPerSocket:   2,
		Memory:              8589934592,
		MachineManager:      client.BaseObject{ID: "mm-1", Name: "xoa"},
		Host:                client.BaseObject{ID: "host-1", Name: "host-name"},
		Pool:                client.BaseObject{ID: "pool-1", Name: "pool-name"},
	}
	vm.DvdDrive.Name = "dvd-1"
	vm.DvdDrive.Attached = true
	vm.Tools.Detected = true
	vm.Tools.Version = "1.0"
	vm.PVDrivers.Detected = true
	vm.PVDrivers.Version = "1.0"
	vm.PVDrivers.AreUpToDate = true
	vm.ManagementAgent.Detected = true
	vm.Addresses.IPv6 = "fe80::1"
	vm.Addresses.IPv4 = "10.0.0.1"
	return vm
}

func fixtureOpenIaasMachineManager() *client.OpenIaaSMachineManager {
	return &client.OpenIaaSMachineManager{
		ID:        "mm-1",
		Name:      "xoa",
		OSVersion: "8.3",
		OSName:    "XCP-ng",
	}
}

func fixtureBackupOpenIaasBackup() *client.Backup {
	return &client.Backup{
		ID:                      "backup-1",
		InternalID:              "internal-1",
		Mode:                    "delta",
		IsVirtualMachineDeleted: true,
		Size:                    2048,
		Timestamp:               1700000000,
		VirtualMachine:          client.BaseObject{ID: "vm-1", Name: "vm-name"},
		Policy:                  client.BaseObject{ID: "policy-1", Name: "policy-name"},
	}
}

func fixtureBackupOpenIaasPolicy() *client.BackupOpenIaasPolicy {
	policy := &client.BackupOpenIaasPolicy{
		ID:              "policy-1",
		Name:            "policy-name",
		InternalID:      "internal-1",
		Running:         true,
		Mode:            "delta",
		MachineManager:  client.BaseObject{ID: "mm-1", Name: "xoa"},
		VirtualMachines: []string{"vm-1"},
	}
	policy.Schedulers = append(policy.Schedulers, struct {
		TemporarilyDisabled bool
		Retention           int
		Cron                string
		Timezone            string
	}{TemporarilyDisabled: true, Retention: 7, Cron: "0 2 * * *", Timezone: "Europe/Paris"})
	return policy
}

func TestDatasourceFlattenOutputsFitTheirSchemas(t *testing.T) {
	checks := []datasourceFlattenCheck{
		// OpenIaaS singles (Read sets the flatten map per key)
		{"iaas_opensource_pool", dataSourceOpenIaasPool(), "", helpers.FlattenOpenIaaSPool(fixtureOpenIaasPool())},
		{"iaas_opensource_host", dataSourceOpenIaasHost(), "", helpers.FlattenOpenIaaSHost(fixtureOpenIaasHost())},
		{"iaas_opensource_network", dataSourceOpenIaasNetwork(), "", helpers.FlattenOpenIaaSNetwork(fixtureOpenIaasNetwork())},
		{"iaas_opensource_network_adapter", dataSourceOpenIaasNetworkAdapter(), "", helpers.FlattenOpenIaaSNetworkAdapter(fixtureOpenIaasNetworkAdapter())},
		{"iaas_opensource_snapshot", dataSourceOpenIaasSnapshot(), "", helpers.FlattenOpenIaaSSnapshot(fixtureOpenIaasSnapshot())},
		{"iaas_opensource_replication_policy", dataSourceOpenIaasReplicationPolicy(), "", helpers.FlattenOpenIaaSReplicationPolicy(fixtureOpenIaasReplicationPolicy())},
		{"iaas_opensource_storage_repository", dataSourceOpenIaasStorageRepository(), "", helpers.FlattenOpenIaaSStorageRepository(fixtureOpenIaasStorageRepository())},
		{"iaas_opensource_template", dataSourceOpenIaasTemplate(), "", helpers.FlattenOpenIaaSTemplate(fixtureOpenIaasTemplate())},
		{"iaas_opensource_virtual_disk", dataSourceOpenIaasVirtualDisk(), "", helpers.FlattenOpenIaaSVirtualDisk(fixtureOpenIaasVirtualDisk(), "")},
		{"iaas_opensource_virtual_machine", dataSourceOpenIaasVirtualMachine(), "", helpers.FlattenOpenIaaSVirtualMachine(fixtureOpenIaasVirtualMachine())},
		{"iaas_opensource_availability_zone", dataSourceOpenIaasMachineManager(), "", helpers.FlattenOpenIaaSMachineManager(fixtureOpenIaasMachineManager())},
		{"backup_iaas_opensource_backup", dataSourceOpenIaasBackup(), "", helpers.FlattenBackupOpenIaasBackup(fixtureBackupOpenIaasBackup())},
		{"backup_iaas_opensource_policy", dataSourceOpenIaasBackupPolicy(), "", helpers.FlattenBackupOpenIaasPolicy(fixtureBackupOpenIaasPolicy())},

		// OpenIaaS lists (Read sets one computed root list)
		{"iaas_opensource_pools", dataSourceOpenIaasPools(), "pools", helpers.FlattenOpenIaaSPool(fixtureOpenIaasPool())},
		{"iaas_opensource_hosts", dataSourceOpenIaasHosts(), "hosts", helpers.FlattenOpenIaaSHost(fixtureOpenIaasHost())},
		{"iaas_opensource_networks", dataSourceOpenIaasNetworks(), "networks", helpers.FlattenOpenIaaSNetwork(fixtureOpenIaasNetwork())},
		{"iaas_opensource_network_adapters", dataSourceOpenIaasNetworkAdapters(), "network_adapters", helpers.FlattenOpenIaaSNetworkAdapter(fixtureOpenIaasNetworkAdapter())},
		{"iaas_opensource_snapshots", dataSourceOpenIaasSnapshots(), "snapshots", helpers.FlattenOpenIaaSSnapshot(fixtureOpenIaasSnapshot())},
		{"iaas_opensource_replication_policies", dataSourceOpenIaasReplicationPolicies(), "policies", helpers.FlattenOpenIaaSReplicationPolicy(fixtureOpenIaasReplicationPolicy())},
		{"iaas_opensource_storage_repositories", dataSourceOpenIaasStorageRepositories(), "storage_repositories", helpers.FlattenOpenIaaSStorageRepository(fixtureOpenIaasStorageRepository())},
		{"iaas_opensource_templates", dataSourceOpenIaasTemplates(), "templates", helpers.FlattenOpenIaaSTemplate(fixtureOpenIaasTemplate())},
		{"iaas_opensource_virtual_disks", dataSourceOpenIaasVirtualDisks(), "virtual_disks", helpers.FlattenOpenIaaSVirtualDisk(fixtureOpenIaasVirtualDisk(), "")},
		{"iaas_opensource_virtual_machines", dataSourceOpenIaasVirtualMachines(), "virtual_machines", helpers.FlattenOpenIaaSVirtualMachine(fixtureOpenIaasVirtualMachine())},
		{"backup_iaas_opensource_backups", dataSourceOpenIaasBackups(), "backups", helpers.FlattenBackupOpenIaasBackup(fixtureBackupOpenIaasBackup())},
		{"backup_iaas_opensource_policies", dataSourceOpenIaasBackupPolicies(), "policies", helpers.FlattenBackupOpenIaasPolicy(fixtureBackupOpenIaasPolicy())},
	}

	for _, check := range checks {
		t.Run(check.name, func(t *testing.T) {
			assertFlattenFitsSchema(t, check)
		})
	}
}
