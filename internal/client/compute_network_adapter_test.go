package client

import (
	"context"
	"testing"
	"time"

	"github.com/sethvargo/go-retry"
	"github.com/stretchr/testify/require"
)

func TestCompute_NetworkAdapterList(t *testing.T) {
	ctx := context.Background()
	networkAdapters, err := client.Compute().NetworkAdapter().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networkAdapters), 1)

	var found bool
	for _, na := range networkAdapters {
		if na.ID == "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_NetworkAdapterRead(t *testing.T) {
	ctx := context.Background()
	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86")
	require.NoError(t, err)

	expected := &NetworkAdapter{
		ID:               "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86",
		VirtualMachineId: "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		Name:             "Network adapter 1",
		Type:             "VMXNET3",
		MacType:          "ASSIGNED",
		MacAddress:       "00:50:56:85:44:2e",
		Connected:        false,
		AutoConnect:      true,
	}
	require.Equal(t, expected, networkAdapter)
}

func TestNetworkAdapterClient_Create(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-network-adapter",
		DatacenterId:              "ac33c033-693b-4fc5-9196-26df77291dbb",
		HostClusterId:             "083b0ed7-8b0f-4cec-be47-78f48b457e6a",
		DatastoreClusterId:        "1a996110-2746-4725-958f-f6fceef05b32",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)
	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	vm, err := client.Compute().VirtualMachine().Read(ctx, activity.ConcernedItems[0].ID)
	require.NoError(t, err)

	activityId, err = client.Compute().NetworkAdapter().Create(ctx, &CreateNetworkAdapterRequest{
		VirtualMachineId: activity.ConcernedItems[0].ID,
		NetworkId:        "cb5d4885-e112-42e9-9842-db4c8fc78f9b",
		Type:             "VMXNET3",
		MacAddress:       "00:50:57:CB:89:B7",
	})
	require.NoError(t, err)
	activity, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	networkAdapterId := activity.ConcernedItems[0].ID

	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, networkAdapterId)
	require.NoError(t, err)
	require.Equal(
		t,
		&NetworkAdapter{
			ID:               networkAdapterId,
			VirtualMachineId: vm.ID,
			Name:             "Network adapter 1",
			Type:             "VMXNET3",
			MacType:          "MANUAL",
			MacAddress:       "00:50:57:CB:89:B7",
			Connected:        false,
			AutoConnect:      false,
		},
		networkAdapter,
	)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &PowerRequest{
		ID:             vm.ID,
		DatacenterId:   vm.VirtualDatacenterId,
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

	activityId, err = client.Compute().NetworkAdapter().Update(ctx, &UpdateNetworkAdapterRequest{
		ID:           networkAdapterId,
		MacType:      "ASSIGNED",
		NewNetworkId: "cb5d4885-e112-42e9-9842-db4c8fc78f9b",
	})
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	networkAdapter, err = client.Compute().NetworkAdapter().Read(ctx, networkAdapterId)
	require.NoError(t, err)
	require.Equal(t, "ASSIGNED", networkAdapter.MacType)
	require.NotEqual(t, "00:50:57:CB:89:B7", networkAdapter.MacAddress)

	activityId, err = client.Compute().VirtualMachine().Power(ctx, &PowerRequest{
		ID:             vm.ID,
		DatacenterId:   vm.VirtualDatacenterId,
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
