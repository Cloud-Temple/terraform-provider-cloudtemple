package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualDiskList(t *testing.T) {
	ctx := context.Background()
	virtualDisks, err := client.Compute().VirtualDisk().List(ctx, "dba8aea7-7718-4ffb-8932-9acf4c8cc629")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDisks), 1)

	var found bool
	for _, vd := range virtualDisks {
		if vd.ID == "c31307d3-bff1-4374-b431-655cbe68ac24" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDiskRead(t *testing.T) {
	ctx := context.Background()
	virtualDatacenter, err := client.Compute().VirtualDisk().Read(ctx, "c31307d3-bff1-4374-b431-655cbe68ac24")
	require.NoError(t, err)

	expected := &VirtualDisk{
		ID:                  "c31307d3-bff1-4374-b431-655cbe68ac24",
		VirtualMachineId:    "dba8aea7-7718-4ffb-8932-9acf4c8cc629",
		MachineManagerId:    "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e",
		Name:                "Hard disk 1",
		Capacity:            5368709120,
		DiskUnitNumber:      0,
		ControllerBusNumber: 0,
		DatastoreId:         "88fb9089-cf33-47f0-938a-fe792f4a9039",
		DatastoreName:       "ds001-t0001-r-stw1-data13-th3s",
		InstantAccess:       false,
		NativeId:            "03ab2449-c80e-4597-a389-39d1af6e5f45",
		DiskPath:            "[ds001-t0001-r-stw1-data13-th3s] tf-do-not-delete/tf-do-not-delete.vmdk",
		ProvisioningType:    "dynamic",
		DiskMode:            "persistent",
		Editable:            true,
	}
	require.Equal(t, expected, virtualDatacenter)
}

func TestVirtualDiskClient_Create(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-disk",
		DatacenterId:              "7b56f202-83e3-4112-9771-8fb001fbac3e",
		HostClusterId:             "c80c4667-2f2d-4087-852b-995b0d5f1f2e",
		DatastoreClusterId:        "0f3c6809-3f15-42c1-a502-69c80bf7ca8f",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: "0f3c6809-3f15-42c1-a502-69c80bf7ca8f",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	diskId := activity.ConcernedItems[0].ID

	disk, err := client.Compute().VirtualDisk().Read(ctx, diskId)
	require.NoError(t, err)

	// Ignore some fields that change often for the test
	disk.DatastoreId = ""
	disk.DatastoreName = ""
	disk.DiskPath = ""

	require.Equal(
		t,
		&VirtualDisk{
			ID:               diskId,
			VirtualMachineId: vm.ID,
			MachineManagerId: "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e",
			Name:             "Hard disk 1",
			Capacity:         10737418240,
			NativeId:         diskId,
			ProvisioningType: "dynamic",
			DiskMode:         "persistent",
			Editable:         true,
		},
		disk,
	)

	activityId, err = client.Compute().VirtualDisk().Update(ctx, &UpdateVirtualDiskRequest{
		ID:          diskId,
		NewCapacity: 2 * 10737418240,
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	disk, err = client.Compute().VirtualDisk().Read(ctx, diskId)
	require.NoError(t, err)

	// Ignore some fields that change often for the test
	diskPath := disk.DiskPath
	disk.DatastoreId = ""
	disk.DatastoreName = ""
	disk.DiskPath = ""

	require.Equal(
		t,
		&VirtualDisk{
			ID:               diskId,
			VirtualMachineId: vm.ID,
			MachineManagerId: "8afdb4e8-b68d-4bb8-a606-3dc47cc2da0e",
			Name:             "Hard disk 1",
			Capacity:         21474836480,
			NativeId:         diskId,
			ProvisioningType: "dynamic",
			DiskMode:         "persistent",
			Editable:         true,
		},
		disk,
	)

	activityId, err = client.Compute().VirtualDisk().Mount(ctx, vm.ID, diskPath)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Delete(ctx, diskId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestVirtualDiskClient_Unmount(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-disk-unmount",
		DatacenterId:              "7b56f202-83e3-4112-9771-8fb001fbac3e",
		HostClusterId:             "c80c4667-2f2d-4087-852b-995b0d5f1f2e",
		DatastoreClusterId:        "0f3c6809-3f15-42c1-a502-69c80bf7ca8f",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: "0f3c6809-3f15-42c1-a502-69c80bf7ca8f",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	diskId := activity.ConcernedItems[0].ID

	activityId, err = client.Compute().VirtualDisk().Unmount(ctx, diskId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
