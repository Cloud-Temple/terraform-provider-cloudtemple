package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	DataStoreDataCenterId = "COMPUTE_DATASTORE_DATACENTER_ID"
	DataStoreMoRef        = "COMPUTE_DATASTORE_MOREF"
	DataStoreUniqueId     = "COMPUTE_DATASTORE_UNIQUE_ID"
	DataStoreType         = "COMPUTE_DATASTORE_TYPE"
	DataStoreHostMoRefs   = "COMPUTE_DATASTORE_HOST_MOREFS"
)

func TestCompute_DatastoreList(t *testing.T) {
	ctx := context.Background()
	datastores, err := client.Compute().Datastore().List(ctx, &DatastoreFilter{
		DatacenterId: os.Getenv(DataStoreDataCenterId),
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(datastores), 1)

	var found bool
	for _, dc := range datastores {
		if dc.ID == os.Getenv(DataStoreId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_DatastoreRead(t *testing.T) {
	ctx := context.Background()
	datastore, err := client.Compute().Datastore().Read(ctx, os.Getenv(DataStoreId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(DataStoreId), datastore.ID)
	require.Equal(t, os.Getenv(DataStoreName), datastore.Name)
	require.Equal(t, os.Getenv(DataStoreMoRef), datastore.Moref)
	require.Equal(t, os.Getenv(DataStoreType), datastore.Type)
	require.Equal(t, os.Getenv(DataStoreUniqueId), datastore.UniqueId)
	require.Equal(t, os.Getenv(MachineManagerId), datastore.MachineManagerId)
}
