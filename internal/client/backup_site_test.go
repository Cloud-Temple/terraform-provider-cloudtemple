package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupSiteClient_List(t *testing.T) {
	ctx := context.Background()
	backupSites, err := client.Backup().Site().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(backupSites), 1)

	var backupSite *BackupSite
	for _, bs := range backupSites {
		if bs.ID == "98e75cf9-6b3c-4422-8d4e-826a032c2bf1" {
			backupSite = bs
			break
		}
	}
	require.NotNil(t, backupSite)

	expected := &BackupSite{
		ID:   "98e75cf9-6b3c-4422-8d4e-826a032c2bf1",
		Name: "DC-TH3S",
	}
	require.Equal(t, expected, backupSite)
}
