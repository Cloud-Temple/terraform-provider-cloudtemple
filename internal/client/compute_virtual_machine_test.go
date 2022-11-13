package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualMachineList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualMachines, err := client.Compute().VirtualMachine().List(ctx, true, "", false, false, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualMachines), 1)

	var found bool
	for _, vm := range virtualMachines {
		if vm.ID == "de2b8b80-8b90-414a-bc33-e12f61a4c05c" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualMachineRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualMachine, err := client.Compute().VirtualMachine().Read(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)

	// Skip checking the storage
	virtualMachine.Storage = VirtualMachineStorage{}

	expected := &VirtualMachine{
		ID:                             "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		Name:                           "virtual_machine_67_bob-clone",
		Moref:                          "vm-1148",
		MachineManagerType:             "vcenter",
		MachineManagerId:               "9dba240e-a605-4103-bac7-5336d3ffd124",
		MachineManagerName:             "vc-vstack-080-bob",
		DatastoreName:                  "ds001-bob-svc1-data4-eqx6",
		ConsolidationNeeded:            false,
		Template:                       false,
		PowerState:                     "running",
		HardwareVersion:                "vmx-14",
		NumCoresPerSocket:              1,
		OperatingSystemName:            "Ubuntu Linux (64-bit)",
		Cpu:                            1,
		CpuHotAddEnabled:               false,
		CpuHotRemoveEnabled:            false,
		MemoryHotAddEnabled:            false,
		Memory:                         1073741824,
		CpuUsage:                       0,
		MemoryUsage:                    10485760,
		Tools:                          "toolsNotInstalled",
		ToolsVersion:                   0,
		VirtualDatacenterId:            "85d53d08-0fa9-491e-ab89-90919516df25",
		DistributedVirtualPortGroupIds: []string{},
		SppMode:                        "production",
		Snapshoted:                     false,
		TriggeredAlarms:                []string{},
		ReplicationConfig: VirtualMachineReplicationConfig{
			Generation:            31,
			VmReplicationId:       "GID-37249b75-7b18-4e33-bdbc-9c96774b7a71",
			Rpo:                   90,
			QuiesceGuestEnabled:   false,
			Paused:                false,
			OppUpdatesEnabled:     false,
			NetCompressionEnabled: false,
			NetEncryptionEnabled:  false,
			EncryptionDestination: false,
			Disk: []VirtualMachineDisk{
				{
					Key:               2000,
					DiskReplicationId: "RDID-63eae7f1-e172-4494-a160-eff1c0d001cf",
				},
			},
		},
		ExtraConfig: []VirtualMachineExtraConfig{
			{Key: "tools.guest.desktop.autolock", Value: "FALSE"},
			{Key: "nvram", Value: "virtual_machine_67_bob-clone.nvram"},
			{Key: "pciBridge0.present", Value: "TRUE"},
			{Key: "svga.present", Value: "TRUE"},
			{Key: "pciBridge4.present", Value: "TRUE"},
			{Key: "pciBridge4.virtualDev", Value: "pcieRootPort"},
			{Key: "pciBridge4.functions", Value: "8"},
			{Key: "pciBridge5.present", Value: "TRUE"},
			{Key: "pciBridge5.virtualDev", Value: "pcieRootPort"},
			{Key: "pciBridge5.functions", Value: "8"},
			{Key: "pciBridge6.present", Value: "TRUE"},
			{Key: "pciBridge6.virtualDev", Value: "pcieRootPort"},
			{Key: "pciBridge6.functions", Value: "8"},
			{Key: "pciBridge7.present", Value: "TRUE"},
			{Key: "pciBridge7.virtualDev", Value: "pcieRootPort"},
			{Key: "pciBridge7.functions", Value: "8"},
			{Key: "hpet0.present", Value: "TRUE"},
			{Key: "sched.cpu.latencySensitivity", Value: "normal"},
			{Key: "ethernet0.pciSlotNumber", Value: "160"},
			{Key: "monitor.phys_bits_used", Value: "43"},
			{Key: "numa.autosize.cookie", Value: "10001"},
			{Key: "numa.autosize.vcpu.maxPerVirtualNode", Value: "1"},
			{Key: "pciBridge0.pciSlotNumber", Value: "17"},
			{Key: "pciBridge4.pciSlotNumber", Value: "21"},
			{Key: "pciBridge5.pciSlotNumber", Value: "22"},
			{Key: "pciBridge6.pciSlotNumber", Value: "23"},
			{Key: "pciBridge7.pciSlotNumber", Value: "24"},
			{Key: "sata0.pciSlotNumber", Value: "33"},
			{Key: "scsi0.pciSlotNumber", Value: "16"},
			{Key: "softPowerOff", Value: "FALSE"},
			{Key: "vmci0.pciSlotNumber", Value: "32"},
			{Key: "vmotion.checkpointFBSize", Value: "4194304"},
			{Key: "vmotion.checkpointSVGAPrimarySize", Value: "4194304"},
			{Key: "toolsInstallManager.lastInstallError", Value: "21004"},
			{Key: "tools.remindInstall", Value: "TRUE"},
			{Key: "toolsInstallManager.updateCounter", Value: "1"},
			{Key: "sched.swap.derivedName", Value: "/vmfs/volumes/601d9902-28268458-ac1a-0025b553004c/virtual_machine_67_bob-clone/virtual_machine_67_bob-clone-0af0ba93.vswp"},
			{Key: "scsi0:0.redo", Value: "{}"},
			{Key: "scsi0:0.hbr_filter.rdid", Value: "RDID-63eae7f1-e172-4494-a160-eff1c0d001cf"},
			{Key: "scsi0:0.hbr_filter.persistent", Value: "hbr-persistent-state-RDID-63eae7f1-e172-4494-a160-eff1c0d001cf.psf"},
			{Key: "scsi0:0.filters", Value: "hbr_filter"},
			{Key: "hbr_filter.configGen", Value: "31"},
			{Key: "hbr_filter.gid", Value: "GID-37249b75-7b18-4e33-bdbc-9c96774b7a71"},
			{Key: "hbr_filter.destination", Value: "10.1.1.178"},
			{Key: "hbr_filter.port", Value: "31031"},
			{Key: "hbr_filter.rpo", Value: "90"},
			{Key: "vmware.tools.internalversion", Value: "0"},
			{Key: "vmware.tools.requiredversion", Value: "12288"},
			{Key: "migrate.hostLogState", Value: "none"},
			{Key: "migrate.migrationId", Value: "0"},
			{Key: "migrate.hostLog", Value: "virtual_machine_67_bob-clone-765a8751.hlog"},
		},
		BootOptions: VirtualMachineBootOptions{
			Firmware:         "bios",
			BootDelay:        0,
			EnterBIOSSetup:   false,
			BootRetryEnabled: false,
			BootRetryDelay:   10000,
		},
	}
	require.Equal(t, expected, virtualMachine)
}

func TestCompute_VirtualMachineCreateDelete(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client",
		DatacenterId:              "85d53d08-0fa9-491e-ab89-90919516df25",
		HostClusterId:             "dde72065-60f4-4577-836d-6ea074384d62",
		DatastoreClusterId:        "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)
}

func TestCompute_UpdateAndPower(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-power",
		DatacenterId:              "85d53d08-0fa9-491e-ab89-90919516df25",
		HostClusterId:             "dde72065-60f4-4577-836d-6ea074384d62",
		DatastoreClusterId:        "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	vm, err := client.Compute().VirtualMachine().Read(ctx, instanceId)
	require.NoError(t, err)
	require.Equal(t, "stopped", vm.PowerState)

	activityId, err = client.Compute().VirtualMachine().Update(ctx, &UpdateVirtualMachineRequest{
		Id: instanceId,
		BootOptions: &BootOptions{
			BootDelay:        0,
			BootRetryDelay:   10000,
			BootRetryEnabled: false,
			EnterBIOSSetup:   false,
			Firmware:         "bios",
		},
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &PowerRequest{
		ID:           instanceId,
		DatacenterId: vm.VirtualDatacenterId,
		PowerAction:  "on",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &PowerRequest{
		ID:           instanceId,
		DatacenterId: vm.VirtualDatacenterId,
		PowerAction:  "off",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)
}

func TestVirtualMachineClient_Rename(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-rename",
		DatacenterId:              "85d53d08-0fa9-491e-ab89-90919516df25",
		HostClusterId:             "dde72065-60f4-4577-836d-6ea074384d62",
		DatastoreClusterId:        "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Rename(ctx, activity.ConcernedItems[0].ID, "test-rename")
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)
	require.Equal(t, "test-rename", vm.Name)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)
}
