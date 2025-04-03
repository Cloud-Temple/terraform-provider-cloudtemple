package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	PolicyId     = "BACKUP_POLICY_ID"
	PolicyName   = "BACKUP_POLICY_NAME"
	DataCenterId = "DATACENTER_ID"
)

func TestBackupSLAPolicyClient_List(t *testing.T) {
	ctx := context.Background()
	slaPolicies, err := client.Backup().SLAPolicy().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(slaPolicies), 1)

	var found bool
	for _, sl := range slaPolicies {
		if sl.ID == os.Getenv(PolicyId) {
			found = true
			break
		}
	}
	require.True(t, found)

	slaPolicies, err = client.Backup().SLAPolicy().List(ctx, &clientpkg.BackupSLAPolicyFilter{
		VirtualMachineId: "12345678-1234-5678-1234-567812345678",
	})
	require.NoError(t, err)

	require.Len(t, slaPolicies, 0)
}

func TestBackupSLAPolicyClient_Read(t *testing.T) {
	ctx := context.Background()
	slaPolicy, err := client.Backup().SLAPolicy().Read(ctx, os.Getenv(PolicyId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(PolicyId), slaPolicy.ID)
	require.Equal(t, os.Getenv(PolicyName), slaPolicy.Name)

}

func TestBackupSLAPolicyClient_AssignVirtualMachine(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Compute().VirtualMachine().Create(ctx, &clientpkg.CreateVirtualMachineRequest{
		Name:                      "test-client-assign-vm",
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

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
