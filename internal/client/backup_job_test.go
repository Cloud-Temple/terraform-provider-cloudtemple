package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupJobClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jobs, err := client.Backup().Job().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobs), 1)

	var found bool
	for _, cl := range jobs {
		if cl.ID == "1004" {
			found = true
			break
		}
	}
	require.True(t, found)

	jobs, err = client.Backup().Job().List(ctx, &BackupJobFilter{
		Type: "catalog",
	})
	require.NoError(t, err)

	require.Len(t, jobs, 1)
}

func TestBackupJobClient_Read(t *testing.T) {
	ctx := context.Background()
	job, err := client.Backup().Job().Read(ctx, "1004")
	require.NoError(t, err)

	expected := &BackupJob{
		ID:          "1004",
		Name:        "Hypervisor Inventory",
		DisplayName: "Hypervisor Inventory",
		Type:        "catalog",
		Status:      "IDLE",
		PolicyId:    "1004",
	}
	require.Equal(t, expected, job)
}

func TestBackupJobClient_Run(t *testing.T) {
	ctx := context.Background()

	activityId, err := client.Backup().Job().Run(ctx, &BackupJobRunRequest{
		JobId: "1004",
	})
	require.NoError(t, err)

	_, err = client.Activity().WaitForCompletion(ctx, activityId)
	require.NoError(t, err)

	_, err = client.Backup().Job().WaitForCompletion(ctx, "1004")
	require.NoError(t, err)
}
