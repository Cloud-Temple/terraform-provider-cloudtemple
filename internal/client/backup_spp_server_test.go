package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestTenantId     = "TEST_TENANT_ID"
	SppServerId      = "TEST_BACKUP_SPPSERVER_ID"
	SppServerame     = "TEST_BACKUP_SPPSERVER_NAME"
	SppServerAddress = "TEST_BACKUP_SPPSERVER_ADDRESS"
)

func TestBackupSPPServerClient_List(t *testing.T) {
	ctx := context.Background()
	sppServers, err := client.Backup().SPPServer().List(ctx, os.Getenv(TestTenantId))
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(sppServers), 1)

	var found bool
	for _, spp := range sppServers {
		if spp.ID == os.Getenv(SppServerId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestBackupSPPServerClient_Read(t *testing.T) {
	ctx := context.Background()
	slaPolicy, err := client.Backup().SPPServer().Read(ctx, os.Getenv(SppServerId))
	require.NoError(t, err)

	expected := &BackupSPPServer{
		ID:      os.Getenv(SppServerId),
		Name:    os.Getenv(SppServerame),
		Address: os.Getenv(SppServerAddress),
	}

	require.Equal(t, expected, slaPolicy)
}
