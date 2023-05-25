package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	HostClusterName  = "TEST_HOST_CLUSTER_NAME"
	HostClusterMoRef = "TEST_HOST_CLUSTER_MOREF"
)

func TestCompute_HostClusterList(t *testing.T) {
	ctx := context.Background()
	hostClusters, err := client.Compute().HostCluster().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hostClusters), 1)

	var found bool
	for _, hc := range hostClusters {
		if hc.ID == os.Getenv(HostClusterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_HostClusterRead(t *testing.T) {
	ctx := context.Background()
	hostCluster, err := client.Compute().HostCluster().Read(ctx, os.Getenv(HostClusterId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(HostClusterId), hostCluster.ID)
	require.Equal(t, os.Getenv(HostClusterName), hostCluster.Name)
	require.Equal(t, os.Getenv(HostClusterMoRef), hostCluster.Moref)
}
