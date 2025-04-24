package client

import (
	"context"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestCompute_SnapshotList(t *testing.T) {
	ctx := context.Background()
	_, err := client.Compute().Snapshot().List(ctx, &clientpkg.SnapshotFilter{})
	require.NoError(t, err)
}
