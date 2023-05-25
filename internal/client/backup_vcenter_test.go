package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	TestSppServerId = "TEST_BACKUP_SPPSERVER_ID"
)

func TestBackupVCenterClient_List(t *testing.T) {
	ctx := context.Background()
	vcenters, err := client.Backup().VCenter().List(ctx, os.Getenv(TestSppServerId))
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, vc := range vcenters {
		if vc.ID == os.Getenv(VirtualDatacenterId) {
			found = true
			break
		}
	}
	require.True(t, found)

}
