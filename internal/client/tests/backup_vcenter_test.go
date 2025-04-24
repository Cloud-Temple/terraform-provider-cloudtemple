package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestBackupVCenterClient_List(t *testing.T) {
	ctx := context.Background()
	vcenters, err := client.Backup().VCenter().List(ctx, &clientpkg.BackupVCenterFilter{
		SppServerId: os.Getenv(SppServerId),
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, vc := range vcenters {
		if vc.ID == os.Getenv(VCenterId) {
			found = true
			break
		}
	}
	require.True(t, found)

}
