package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	JobId = "BACKUP_JOB_ID"
)

func TestBackupJobClient_List(t *testing.T) {
	ctx := context.Background()
	jobs, err := client.Backup().Job().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobs), 1)

	var found bool
	for _, cl := range jobs {
		if cl.ID == os.Getenv(JobId) {
			found = true
			break
		}
	}
	require.True(t, found)

	jobs, err = client.Backup().Job().List(ctx, &clientpkg.BackupJobFilter{
		Type: "catalog",
	})
	require.NoError(t, err)

	require.Len(t, jobs, 2)
}

func TestBackupJobClient_Read(t *testing.T) {
	ctx := context.Background()
	job, err := client.Backup().Job().Read(ctx, os.Getenv(JobId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(JobId), job.ID)
	require.Equal(t, "Hypervisor Inventory", job.Name)
	require.Equal(t, "catalog", job.Type)
}

func TestBackupJobClient_Run(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Backup().Job().Run(ctx, &clientpkg.BackupJobRunRequest{
		JobId: os.Getenv(JobId),
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	_, err = client.Backup().Job().WaitForCompletion(ctx, os.Getenv(JobId), nil)
	require.NoError(t, err)
}
