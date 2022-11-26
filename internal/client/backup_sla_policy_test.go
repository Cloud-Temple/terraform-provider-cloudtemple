package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupSLAPolicyClient_List(t *testing.T) {
	ctx := context.Background()
	slaPolicies, err := client.Backup().SLAPolicy().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(slaPolicies), 1)

	var found bool
	for _, sl := range slaPolicies {
		if sl.ID == "442718ef-44a1-43d7-9b57-2d910d74e928" {
			found = true
			break
		}
	}
	require.True(t, found)

	slaPolicies, err = client.Backup().SLAPolicy().List(ctx, &BackupSLAPolicyFilter{
		VirtualMachineId: "12345678-1234-5678-1234-567812345678",
	})
	require.NoError(t, err)

	require.Len(t, slaPolicies, 0)
}

func TestBackupSLAPolicyClient_Read(t *testing.T) {
	ctx := context.Background()
	slaPolicy, err := client.Backup().SLAPolicy().Read(ctx, "442718ef-44a1-43d7-9b57-2d910d74e928")
	require.NoError(t, err)

	expected := &BackupSLAPolicy{
		ID:   "442718ef-44a1-43d7-9b57-2d910d74e928",
		Name: "SLA_ADMIN",
		SubPolicies: []*BackupSLASubPolicy{
			{
				Type:          "REPLICATION",
				UseEncryption: false,
				Software:      true,
				Site:          "DC-EQX6",
				Retention: BackupSLAPolicyRetention{
					Age: 15,
				},
				Trigger: BackupSLAPolicyTrigger{
					Frequency:    1,
					Type:         "DAILY",
					ActivateDate: 1568617200000,
				},
				Target: BackupSLAPolicyTarget{
					ID:           "1000",
					Href:         "https://spp1-ctlabs-eqx6.backup.cloud-temple.lan/api/site/1000",
					ResourceType: "site",
				},
			},
		},
	}

	require.Equal(t, expected, slaPolicy)
}

func TestBackupSLAPolicyClient_AssignVirtualMachine(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &CreateVirtualMachineRequest{
		Name:                      "test-client-assign-vm",
		DatacenterId:              "85d53d08-0fa9-491e-ab89-90919516df25",
		HostClusterId:             "dde72065-60f4-4577-836d-6ea074384d62",
		DatastoreClusterId:        "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		GuestOperatingSystemMoref: "amazonlinux2_64Guest",
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.ConcernedItems[0].ID

	jobs, err := client.Backup().Job().List(ctx, &BackupJobFilter{
		Type: "catalog",
	})
	require.NoError(t, err)
	require.Len(t, jobs, 1)

	activityId, err = client.Backup().Job().Run(ctx, &BackupJobRunRequest{
		JobId: jobs[0].ID,
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	_, err = client.Backup().Job().WaitForCompletion(ctx, jobs[0].ID, nil)
	require.NoError(t, err)

	activityId, err = client.Backup().SLAPolicy().AssignVirtualMachine(ctx, &BackupAssignVirtualMachineRequest{
		VirtualMachineIds: []string{instanceId},
		SLAPolicies:       []string{"442718ef-44a1-43d7-9b57-2d910d74e928"},
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Backup().SLAPolicy().AssignVirtualMachine(ctx, &BackupAssignVirtualMachineRequest{
		VirtualMachineIds: []string{instanceId},
		SLAPolicies:       []string{},
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
