package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	ResourcePoolId    = "COMPTE_RESOURCE_POOL_ID"
	ResourcePoolName  = "COMPTE_RESOURCE_POOL_NAME"
	ResourcePoolMoRef = "COMPTE_RESOURCE_POOL_MOFER"
)

func TestCompute_ResourcePoolList(t *testing.T) {
	ctx := context.Background()
	resourcePools, err := client.Compute().ResourcePool().List(ctx, &clientpkg.ResourcePoolFilter{})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(resourcePools), 1)

	var found bool
	for _, h := range resourcePools {
		if h.ID == os.Getenv(ResourcePoolId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_ResourcePoolRead(t *testing.T) {
	ctx := context.Background()
	resourcePool, err := client.Compute().ResourcePool().Read(ctx, os.Getenv(ResourcePoolId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(ResourcePoolId), resourcePool.ID)
	require.Equal(t, os.Getenv(ResourcePoolName), resourcePool.Name)
	require.Equal(t, os.Getenv(ResourcePoolMoRef), resourcePool.Moref)
	require.Equal(t, os.Getenv(MachineManagerId), resourcePool.MachineManager.ID)
}
