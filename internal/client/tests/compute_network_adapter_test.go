package client

import (
	"context"
	"os"
	"testing"
	"time"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/sethvargo/go-retry"
	"github.com/stretchr/testify/require"
)

const (
	NetworkAdapterId               = "COMPUTE_NETWORK_ADAPTER_ID"
	NetworkAdapterName             = "COMPUTE_NETWORK_ADAPTER_NAME"
	NetworkAdapterType             = "COMPUTE_NETWORK_ADAPTER_TYPE"
	NetworkAdapterVirtualMachineId = "COMPUTE_NETWORK_ADAPTER_VIRTUAL_MACHINE"
)

func TestCompute_NetworkAdapterList(t *testing.T) {
	ctx := context.Background()
	networkAdapters, err := client.Compute().NetworkAdapter().List(ctx, &clientpkg.NetworkAdapterFilter{
		VirtualMachineID: os.Getenv(NetworkAdapterVirtualMachineId),
	})

	require.NoError(t, err)
	require.GreaterOrEqual(t, len(networkAdapters), 1)

	var found bool
	for _, na := range networkAdapters {
		if na.ID == os.Getenv(NetworkAdapterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_NetworkAdapterRead(t *testing.T) {
	ctx := context.Background()
	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, os.Getenv(NetworkAdapterId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(NetworkAdapterId), networkAdapter.ID)
	require.Equal(t, os.Getenv(NetworkAdapterName), networkAdapter.Name)
	require.Equal(t, os.Getenv(NetworkAdapterType), networkAdapter.Type)
	require.Equal(t, "", networkAdapter.NetworkId)
	require.Equal(t, os.Getenv(NetworkAdapterVirtualMachineId), networkAdapter.VirtualMachineId)
}

func TestNetworkAdapterClient_Create(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-network-adapter",
		DatacenterId:              os.Getenv(DatacenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperatingSystemMoref),
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

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, instanceId)
	require.NoError(t, err)

	activityId, err = client.Compute().NetworkAdapter().Create(ctx, &clientpkg.CreateNetworkAdapterRequest{
		VirtualMachineId: instanceId,
		NetworkId:        os.Getenv(NetworkId),
		Type:             os.Getenv(NetworkAdapterType),
		MacAddress:       "00:50:56:86:4a:27",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	networkAdapterId := activity.ConcernedItems[0].ID

	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, networkAdapterId)

	require.NoError(t, err)

	require.Equal(t, networkAdapterId, networkAdapter.ID)
	require.Equal(t, os.Getenv(NetworkAdapterName), networkAdapter.Name)
	require.Equal(t, os.Getenv(NetworkAdapterType), networkAdapter.Type)
	require.Equal(t, os.Getenv(NetworkId), networkAdapter.NetworkId)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &clientpkg.PowerRequest{
		ID:             vm.ID,
		DatacenterId:   vm.Datacenter.ID,
		PowerAction:    "on",
		ForceEnterBIOS: false,
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	// Connecting a network adapter can fail right after the VM has been powered
	// on so we retry this part of the test
	b := retry.NewFibonacci(1 * time.Second)
	b = retry.WithCappedDuration(20*time.Second, b)
	b = retry.WithMaxDuration(2*time.Minute, b)

	err = retry.Do(ctx, b, func(ctx context.Context) error {
		activityId, err = client.Compute().NetworkAdapter().Connect(ctx, networkAdapterId)
		if err != nil {
			return err
		}
		_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
		return err
	})
	require.NoError(t, err)

	activityId, err = client.Compute().NetworkAdapter().Disconnect(ctx, networkAdapterId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().NetworkAdapter().Update(ctx, &clientpkg.UpdateNetworkAdapterRequest{
		ID:           networkAdapterId,
		NewNetworkId: "1a2e7257-0747-474a-ba49-942ee463a94c",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	networkAdapter, err = client.Compute().NetworkAdapter().Read(ctx, networkAdapterId)
	require.NoError(t, err)
	require.Equal(t, "ASSIGNED", networkAdapter.MacType)
	require.NotEqual(t, "00:50:57:CB:89:B7", networkAdapter.MacAddress)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &clientpkg.PowerRequest{
		ID:             vm.ID,
		DatacenterId:   vm.Datacenter.ID,
		PowerAction:    "off",
		ForceEnterBIOS: false,
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, vm.ID)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
