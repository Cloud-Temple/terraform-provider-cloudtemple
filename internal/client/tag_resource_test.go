package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTagClient(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-tags",
		DatacenterId:              "85d53d08-0fa9-491e-ab89-90919516df25",
		HostClusterId:             "dde72065-60f4-4577-836d-6ea074384d62",
		DatastoreClusterId:        "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	tags, err := client.Tag().Resource().Read(ctx, instanceId)
	require.NoError(t, err)

	require.Equal(t, []*Tag{}, tags)

	err = client.Tag().Resource().Create(ctx, &CreateTagRequest{
		Key:   "Test",
		Value: "working",
		Resources: []*CreateTagRequestResource{
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
		[]*Tag{
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
		[]*Tag{
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
