package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupVCenterClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	vcenters, err := client.Backup().VCenter().List(ctx, "a3d46fb5-29af-4b98-a665-1e82a62fd6d3")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(vcenters), 1)

	var found bool
	for _, vc := range vcenters {
		if vc.ID == "9dba240e-a605-4103-bac7-5336d3ffd124" {
			found = true
			break
		}
	}
	require.True(t, found)

}
