package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupJobSessionClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	jobSessions, err := client.Backup().JobSession().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobSessions), 1)

	var jobSession *BackupJobSession
	for _, jb := range jobSessions {
		if jb.ID == "1668240000212" {
			jobSession = jb
			break
		}
	}
	require.NotNil(t, jobSession)

	expected := &BackupJobSession{
		ID:            "1668240000212",
		JobName:       "vmware_SLA_CATALOG_SPP",
		SlaPolicyType: "protection",
		JobId:         "1166",
		Type:          "protection",
		Duration:      69102,
		Start:         1668240001081,
		End:           1668240070183,
		Status:        "FAILED",
		Statistics: BackupStatistics{
			Total:   1,
			Success: 0,
			Failed:  1,
			Skipped: 0,
		},
		SLAPolicies: []*BackupSLAPolicyStub{
			{
				ID:   "2115",
				Name: "SLA_CATALOG_SPP",
				HREF: "https://spp1-ctlabs-eqx6.backup.cloud-temple.lan/api/spec/storageprofile/2115",
			},
		},
	}
	require.Equal(t, expected, jobSession)
}
