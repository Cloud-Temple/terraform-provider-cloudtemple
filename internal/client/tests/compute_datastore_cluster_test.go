package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	DatastoreClusterId    = "COMPUTE_DATASTORE_CLUSTER_ID"
	DatastoreClusterName  = "COMPUTE_DATASTORE_CLUSTER_NAME"
	DatastoreClusterMoRef = "COMPUTE_DATASTORE_CLUSTER_MOREF"
	MachineManagerId2     = "COMPUTE_VCENTER_ID_2"
)

func TestCompute_DatastoreClusterList(t *testing.T) {
	ctx := context.Background()
	datastoreClusters, err := client.Compute().DatastoreCluster().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(datastoreClusters), 1)

	var found bool
	for _, dc := range datastoreClusters {
		if dc.ID == os.Getenv(DatastoreClusterId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_DatastoreClusterRead(t *testing.T) {
	ctx := context.Background()
	datastoreCluster, err := client.Compute().DatastoreCluster().Read(ctx, os.Getenv(DatastoreClusterId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(DatastoreClusterId), datastoreCluster.ID)
	require.Equal(t, os.Getenv(DatastoreClusterName), datastoreCluster.Name)
	require.Equal(t, os.Getenv(DatastoreClusterMoRef), datastoreCluster.Moref)
	require.Equal(t, os.Getenv(MachineManagerId2), datastoreCluster.MachineManager.ID)
}
