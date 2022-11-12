package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupSiteClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	backupSites, err := client.Backup().Site().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(backupSites), 1)

	var backupSite *BackupSite
	for _, bs := range backupSites {
		if bs.ID == "7e76f68f-3392-401b-9c3a-d504af96643d" {
			backupSite = bs
			break
		}
	}
	require.NotNil(t, backupSite)

	expected := &BackupSite{
		ID:   "7e76f68f-3392-401b-9c3a-d504af96643d",
		Name: "DC-EQX6",
	}
	require.Equal(t, expected, backupSite)
}
