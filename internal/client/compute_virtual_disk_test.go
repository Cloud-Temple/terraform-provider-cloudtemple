package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualDiskList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualDisks, err := client.Compute().VirtualDisk().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDisks), 1)

	var found bool
	for _, vd := range virtualDisks {
		if vd.ID == "d370b8cd-83eb-4315-a5d9-42157e2e4bb4" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDiskRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualDatacenter, err := client.Compute().VirtualDisk().Read(ctx, "d370b8cd-83eb-4315-a5d9-42157e2e4bb4")
	require.NoError(t, err)

	expected := &VirtualDisk{
		ID:                  "d370b8cd-83eb-4315-a5d9-42157e2e4bb4",
		VirtualMachineId:    "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		MachineManagerId:    "9dba240e-a605-4103-bac7-5336d3ffd124",
		Name:                "Hard disk 1",
		Capacity:            17179869184,
		DiskUnitNumber:      0,
		ControllerBusNumber: 0,
		DatastoreId:         "d439d467-943a-49f5-a022-c0c25b737022",
		DatastoreName:       "ds001-bob-svc1-data4-eqx6",
		InstantAccess:       false,
		NativeId:            "6000C296-f0b9-c149-21f6-a1877fc8bae8",
		DiskPath:            "[ds001-bob-svc1-data4-eqx6] virtual_machine_67_bob-clone/virtual_machine_67_bob-clone_2.vmdk",
		ProvisioningType:    "staticDiffered",
		DiskMode:            "persistent",
		Editable:            true,
	}
	require.Equal(t, expected, virtualDatacenter)
}

func TestVirtualDiskClient_Create(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-disk",
		DatacenterId:              "ac33c033-693b-4fc5-9196-26df77291dbb",
		HostClusterId:             "083b0ed7-8b0f-4cec-be47-78f48b457e6a",
		DatastoreClusterId:        "1a996110-2746-4725-958f-f6fceef05b32",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: "1a996110-2746-4725-958f-f6fceef05b32",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	diskId := activity.ConcernedItems[0].ID

	disk, err := client.Compute().VirtualDisk().Read(ctx, diskId)
	require.NoError(t, err)
	require.Equal(
		t,
		&VirtualDisk{
			ID:                  diskId,
			VirtualMachineId:    vm.ID,
			MachineManagerId:    "9dba240e-a605-4103-bac7-5336d3ffd124",
			Name:                "Hard disk 1",
			Capacity:            10737418240,
			DiskUnitNumber:      0,
			ControllerBusNumber: 0,
			DatastoreId:         "24371f16-b480-40d3-9587-82f97933abca",
			DatastoreName:       "ds002-bob-svc1-stor4-th3",
			InstantAccess:       false,
			NativeId:            diskId,
			DiskPath:            "[ds002-bob-svc1-stor4-th3] test-client-disk_2/test-client-disk.vmdk",
			ProvisioningType:    "dynamic",
			DiskMode:            "persistent",
			Editable:            true,
		},
		disk,
	)

	activityId, err = client.Compute().VirtualDisk().Update(ctx, &UpdateVirtualDiskRequest{
		ID:          diskId,
		NewCapacity: 2 * 10737418240,
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	disk, err = client.Compute().VirtualDisk().Read(ctx, diskId)
	require.NoError(t, err)
	require.Equal(
		t,
		&VirtualDisk{
			ID:                  diskId,
			VirtualMachineId:    vm.ID,
			MachineManagerId:    "9dba240e-a605-4103-bac7-5336d3ffd124",
			Name:                "Hard disk 1",
			Capacity:            21474836480,
			DiskUnitNumber:      0,
			ControllerBusNumber: 0,
			DatastoreId:         "24371f16-b480-40d3-9587-82f97933abca",
			DatastoreName:       "ds002-bob-svc1-stor4-th3",
			InstantAccess:       false,
			NativeId:            diskId,
			DiskPath:            "[ds002-bob-svc1-stor4-th3] test-client-disk_2/test-client-disk.vmdk",
			ProvisioningType:    "dynamic",
			DiskMode:            "persistent",
			Editable:            true,
		},
		disk,
	)

	// activityId, err = client.Compute().VirtualDisk().Unmount(ctx, diskId)
	// require.NoError(t, err)
	// _, err = client.Activity().WaitForCompletion(ctx, activityId)
	// require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Mount(ctx, vm.ID, "[ds002-bob-svc1-stor4-th3] test-client-disk_2/test-client-disk.vmdk")
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Delete(ctx, diskId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)
}

func TestVirtualDiskClient_Unmount(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-disk-unmount",
		DatacenterId:              "ac33c033-693b-4fc5-9196-26df77291dbb",
		HostClusterId:             "083b0ed7-8b0f-4cec-be47-78f48b457e6a",
		DatastoreClusterId:        "1a996110-2746-4725-958f-f6fceef05b32",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: "1a996110-2746-4725-958f-f6fceef05b32",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	diskId := activity.ConcernedItems[0].ID

	activityId, err = client.Compute().VirtualDisk().Unmount(ctx, diskId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)
}
