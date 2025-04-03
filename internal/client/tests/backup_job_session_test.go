package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	JobSessionId   = "BACKUP_JOB_SESSION_ID"
	JobSessionName = "BACKUP_JOB_SESSION_NAME"
)

func TestBackupJobSessionClient_List(t *testing.T) {
	ctx := context.Background()
	jobSessions, err := client.Backup().JobSession().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobSessions), 1)

	var jobSession *clientpkg.BackupJobSession
	for _, jb := range jobSessions {
		if jb.ID == os.Getenv(JobSessionId) {
			jobSession = jb
			break
		}
	}
	require.NotNil(t, jobSession)

	require.Equal(t, os.Getenv(JobSessionId), jobSession.ID)
	require.Equal(t, os.Getenv(JobSessionName), jobSession.JobName)
	require.Equal(t, os.Getenv(JobId), jobSession.JobId)
	require.Equal(t, os.Getenv(JobType), jobSession.Type)

}
