package client

import (
	"context"
	"fmt"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	VirtualMachineId                    = "COMPUTE_VIRTUAL_MACHINE_ID"
	VirtualMachineName                  = "COMPUTE_VIRTUAL_MACHINE_NAME"
	VirtualMachineMoref                 = "COMPUTE_VIRTUAL_MACHINE_MOREF"
	VirtualMachineIdAlternative         = "COMPUTE_VIRTUAL_MACHINE_ID_ALTERNATIVE"
	VirtualMachineHostClusterIdRelocate = "COMPUTE_VIRTUAL_MACHINE_HOST_CLUSTER_RELOCATE"
)

func TestCompute_VirtualMachineList(t *testing.T) {
	ctx := context.Background()
	virtualMachines, err := client.Compute().VirtualMachine().List(ctx, true, "", false, false, nil, nil, nil, nil, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualMachines), 1)

	var found bool
	for _, vm := range virtualMachines {
		if vm.ID == os.Getenv(VirtualMachineId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualMachineRead(t *testing.T) {
	ctx := context.Background()
	virtualMachine, err := client.Compute().VirtualMachine().Read(ctx, os.Getenv(VirtualMachineId))
	require.NoError(t, err)
	require.Equal(t, os.Getenv(VirtualMachineId), virtualMachine.ID)
	require.Equal(t, os.Getenv(VirtualMachineName), virtualMachine.Name)
	require.Equal(t, os.Getenv(VirtualMachineMoref), virtualMachine.Moref)
	require.Equal(t, os.Getenv(MachineManagerId2), virtualMachine.MachineManagerId)
	require.Equal(t, os.Getenv(VirtualMachineMoref), virtualMachine.Moref)

}

func TestCompute_VirtualMachineCreateDelete(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestCompute_UpdateAndPower(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-power",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	jobs, err := client.Backup().Job().List(ctx, &clientpkg.BackupJobFilter{
		Type: "catalog",
	})
	require.NoError(t, err)
	require.Greater(t, len(jobs), 0)

	var job = &clientpkg.BackupJob{}
	for _, currJob := range jobs {
		if currJob.Name == "Hypervisor Inventory" {
			job = currJob
		}
	}

	activityId, err = client.Backup().Job().Run(ctx, &clientpkg.BackupJobRunRequest{
		JobId: job.ID,
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	_, err = client.Backup().Job().WaitForCompletion(ctx, jobs[0].ID, nil)
	require.NoError(t, err)

	activityId, err = client.Backup().SLAPolicy().AssignVirtualMachine(ctx, &clientpkg.BackupAssignVirtualMachineRequest{
		VirtualMachineIds: []string{instanceId},
		SLAPolicies:       []string{os.Getenv(PolicyId)},
	})
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, instanceId)
	require.NoError(t, err)
	require.Equal(t, "stopped", vm.PowerState)

	activityId, err = client.Compute().VirtualMachine().Update(ctx, &clientpkg.UpdateVirtualMachineRequest{
		Id: instanceId,
		BootOptions: &clientpkg.BootOptions{
			BootDelay:        0,
			BootRetryDelay:   10000,
			BootRetryEnabled: false,
			EnterBIOSSetup:   false,
			Firmware:         "bios",
		},
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &clientpkg.PowerRequest{
		ID:           instanceId,
		DatacenterId: vm.DatacenterId,
		PowerAction:  "on",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &clientpkg.PowerRequest{
		ID:           instanceId,
		DatacenterId: vm.DatacenterId,
		PowerAction:  "off",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestVirtualMachineClient_Rename(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-rename",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Rename(ctx, activity.ConcernedItems[0].ID, "test-client-rename-success")
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)
	require.Equal(t, "test-client-rename-success", vm.Name)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestVirtualMachineClient_Clone(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-clone",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})

	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	activityId, err = client.Compute().VirtualMachine().Clone(ctx, &clientpkg.CloneVirtualMachineRequest{
		Name:              "test-client-cloned",
		VirtualMachineId:  instanceId,
		DatacenterId:      os.Getenv(DataCenterId),
		HostClusterId:     os.Getenv(HostClusterId),
		DatatoreClusterId: os.Getenv(DatastoreClusterId),
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.State["completed"].Result)
	require.NoError(t, err)
	require.Equal(t, "test-client-cloned", vm.Name)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestVirtualMachineClient_Relocate(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-relocate",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	activityId, err = client.Compute().VirtualMachine().Relocate(ctx, &clientpkg.RelocateVirtualMachineRequest{
		VirtualMachines:    []string{instanceId},
		Priority:           "highPriority",
		DatacenterId:       os.Getenv(DataCenterId),
		HostClusterId:      os.Getenv(VirtualMachineHostClusterIdRelocate),
		DatastoreClusterId: os.Getenv(DatastoreClusterId),
	})

	newInstanceId := activity.ConcernedItems[0].ID
	fmt.Println(activity.ConcernedItems[0])
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, newInstanceId)
	require.NoError(t, err)
	require.Equal(t, os.Getenv(DataCenterId), vm.DatacenterId)
	require.Equal(t, os.Getenv(VirtualMachineHostClusterIdRelocate), vm.HostClusterId)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, newInstanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}

func TestVirtualMachineClient_Guest(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-clone",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	activityId, err = client.Compute().VirtualMachine().Guest(ctx, instanceId, &clientpkg.UpdateGuestRequest{
		GuestOperatingSystemMoref: "vmwarePhoton64Guest",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, instanceId)
	require.NoError(t, err)
	require.Equal(t, "vmwarePhoton64Guest", vm.OperatingSystemMoref)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
