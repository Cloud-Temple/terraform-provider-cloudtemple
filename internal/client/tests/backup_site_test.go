package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	SiteId   = "BACKUP_SITE_ID"
	SiteName = "BACKUP_SITE_NAME"
)

func TestBackupSiteClient_List(t *testing.T) {
	ctx := context.Background()
	backupSites, err := client.Backup().Site().List(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(backupSites), 1)

	var backupSite *clientpkg.BackupSite
	for _, bs := range backupSites {
		if bs.ID == os.Getenv(SiteId) {
			backupSite = bs
			break
		}
	}
	require.NotNil(t, backupSite)

	expected := &clientpkg.BackupSite{
		ID:   os.Getenv(SiteId),
		Name: os.Getenv(SiteName),
	}
	require.Equal(t, expected, backupSite)
}
