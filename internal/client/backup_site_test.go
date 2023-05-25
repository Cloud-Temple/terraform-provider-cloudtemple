package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	SiteId   = "TEST_BACKUP_SITE_ID"
	SiteName = "TEST_BACKUP_SITE_NAME"
)

func TestBackupSiteClient_List(t *testing.T) {
	ctx := context.Background()
	backupSites, err := client.Backup().Site().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(backupSites), 1)

	var backupSite *BackupSite
	for _, bs := range backupSites {
		if bs.ID == os.Getenv(SiteId) {
			backupSite = bs
			break
		}
	}
	require.NotNil(t, backupSite)

	expected := &BackupSite{
		ID:   os.Getenv(SiteId),
		Name: os.Getenv(SiteName),
	}
	require.Equal(t, expected, backupSite)
}
