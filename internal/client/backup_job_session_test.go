package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupJobSessionClient_List(t *testing.T) {
	ctx := context.Background()
	jobSessions, err := client.Backup().JobSession().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobSessions), 1)

	var jobSession *BackupJobSession
	for _, jb := range jobSessions {
		if jb.ID == "1681977600058" {
			jobSession = jb
			break
		}
	}
	require.NotNil(t, jobSession)

	expected := &BackupJobSession{
		ID:            "1681977600058",
		JobName:       "catalog_SLA_CATALOG_SPP",
		SlaPolicyType: "protection",
		JobId:         "1003",
		Type:          "protection",
		Duration:      23036,
		Start:         1681977600551,
		End:           1681977623587,
		Status:        "COMPLETED",
		SLAPolicies: []*BackupSLAPolicyStub{
			{
				ID:   "2103",
				Name: "SLA_CATALOG_SPP",
				HREF: "https://10.12.8.1/api/spec/storageprofile/2103",
			},
		},
	}
	require.Equal(t, expected, jobSession)
}
