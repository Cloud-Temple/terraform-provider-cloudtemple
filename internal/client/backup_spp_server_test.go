package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupSPPServerClient_List(t *testing.T) {
	ctx := context.Background()
	sppServers, err := client.Backup().SPPServer().List(ctx, "e225dbf8-e7c5-4664-a595-08edf3526080")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(sppServers), 1)

	var found bool
	for _, spp := range sppServers {
		if spp.ID == "a3d46fb5-29af-4b98-a665-1e82a62fd6d3" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestBackupSPPServerClient_Read(t *testing.T) {
	ctx := context.Background()
	slaPolicy, err := client.Backup().SPPServer().Read(ctx, "a3d46fb5-29af-4b98-a665-1e82a62fd6d3")
	require.NoError(t, err)

	expected := &BackupSPPServer{
		ID:      "a3d46fb5-29af-4b98-a665-1e82a62fd6d3",
		Name:    "10",
		Address: "10.1.11.32",
	}

	require.Equal(t, expected, slaPolicy)
}
