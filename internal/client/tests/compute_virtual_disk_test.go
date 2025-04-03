package client

import (
	"context"
	"os"
	"strconv"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	VirtualDiskId          = "COMPUTE_VIRTUAL_DISK_ID"
	VirtualDiskName        = "COMPUTE_VIRTUAL_DISK_NAME"
	VirtualMachineCapacity = "COMPUTE_VIRTUAL_DISK_CAPACITY"
)

func TestCompute_VirtualDiskList(t *testing.T) {
	ctx := context.Background()
	virtualDisks, err := client.Compute().VirtualDisk().List(ctx, os.Getenv(VirtualMachineIdAlternative))
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDisks), 1)

	var found bool
	for _, vd := range virtualDisks {
		if vd.ID == os.Getenv(VirtualDiskId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDiskRead(t *testing.T) {
	ctx := context.Background()
	virtualDisk, err := client.Compute().VirtualDisk().Read(ctx, os.Getenv(VirtualDiskId))
	require.NoError(t, err)

	capacity, _ := strconv.Atoi(os.Getenv(VirtualMachineCapacity))
	require.Equal(t, os.Getenv(VirtualDiskId), virtualDisk.ID)
	require.Equal(t, os.Getenv(VirtualMachineIdAlternative), virtualDisk.VirtualMachineId)
	require.Equal(t, os.Getenv(MachineManagerId2), virtualDisk.MachineManagerId)
	require.Equal(t, capacity, virtualDisk.Capacity)
}

func TestVirtualDiskClient_Create(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-disk",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &clientpkg.CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: os.Getenv(DatastoreClusterId),
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
		&clientpkg.VirtualDisk{
			ID:               diskId,
			VirtualMachineId: vm.ID,
			MachineManagerId: os.Getenv(MachineManagerId2),
			Name:             os.Getenv(VirtualDiskName),
			Capacity:         10737418240,
			NativeId:         diskId,
			ProvisioningType: "dynamic",
			DiskMode:         "persistent",
			Editable:         true,
		},
		disk,
	)

	activityId, err = client.Compute().VirtualDisk().Update(ctx, &clientpkg.UpdateVirtualDiskRequest{
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
		&clientpkg.VirtualDisk{
			ID:               diskId,
			VirtualMachineId: vm.ID,
			MachineManagerId: os.Getenv(MachineManagerId2),
			Name:             os.Getenv(VirtualDiskName),
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
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-disk-unmount",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualDisk().Create(ctx, &clientpkg.CreateVirtualDiskRequest{
		ProvisioningType:   "dynamic",
		DiskMode:           "persistent",
		Capacity:           10737418240,
		VirtualMachineId:   vm.ID,
		DatastoreClusterId: os.Getenv(DatastoreClusterId),
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
