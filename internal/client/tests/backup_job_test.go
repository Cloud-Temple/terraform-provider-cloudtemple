package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	JobId       = "BACKUP_JOB_ID"
	JobName     = "BACKUP_JOB_NAME"
	JobType     = "BACKUP_JOB_TYPE"
	JobPolicyId = "BACKUP_JOB_POLICY_ID"
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
		Type: os.Getenv(JobType),
	})
	require.NoError(t, err)

	require.Len(t, jobs, 2)
}

func TestBackupJobClient_Read(t *testing.T) {
	ctx := context.Background()
	job, err := client.Backup().Job().Read(ctx, os.Getenv(JobId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(JobId), job.ID)
	require.Equal(t, os.Getenv(JobName), job.Name)
	require.Equal(t, os.Getenv(JobType), job.Type)
	require.Equal(t, os.Getenv(JobPolicyId), job.PolicyId)
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
