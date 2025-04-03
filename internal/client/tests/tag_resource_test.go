package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestTagClient(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-tags",
		DatacenterId:              os.Getenv(DataCenterId),
		HostClusterId:             os.Getenv(HostClusterId),
		DatastoreClusterId:        os.Getenv(DatastoreClusterId),
		GuestOperatingSystemMoref: os.Getenv(OperationSystemMoref),
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	tags, err := client.Tag().Resource().Read(ctx, instanceId)
	require.NoError(t, err)

	require.Equal(t, []*clientpkg.Tag{}, tags)

	err = client.Tag().Resource().Create(ctx, &clientpkg.CreateTagRequest{
		Key:   "Test",
		Value: "working",
		Resources: []*clientpkg.CreateTagRequestResource{
			{
				UUID:   instanceId,
				Type:   "vcenter_virtual_machine",
				Source: "vmware",
			},
		},
	})
	require.NoError(t, err)

	tags, err = client.Tag().Resource().Read(ctx, instanceId)
	require.NoError(t, err)

	require.Equal(
		t,
		[]*clientpkg.Tag{
			{
				Key:      "Test",
				Value:    "working",
				Tenant:   testTenantID(t),
				Resource: instanceId,
			},
		},
		tags,
	)

	err = client.Tag().Resource().Delete(ctx, instanceId, "Test")
	require.NoError(t, err)

	require.Equal(
		t,
		[]*clientpkg.Tag{
			{
				Key:      "Test",
				Value:    "working",
				Tenant:   testTenantID(t),
				Resource: instanceId,
			},
		},
		tags,
	)

	tags, err = client.Tag().Resource().Read(ctx, instanceId)
	require.NoError(t, err)

	require.Len(t, tags, 0)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
