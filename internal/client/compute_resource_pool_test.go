package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_ResourcePoolList(t *testing.T) {
	ctx := context.Background()
	resourcePools, err := client.Compute().ResourcePool().List(ctx, "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(resourcePools), 1)

	var found bool
	for _, h := range resourcePools {
		if h.ID == "d21f84fd-5063-4383-b2b0-65b9f25eac27" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_ResourcePoolRead(t *testing.T) {
	ctx := context.Background()
	resourcePool, err := client.Compute().ResourcePool().Read(ctx, "d21f84fd-5063-4383-b2b0-65b9f25eac27")
	require.NoError(t, err)

	// ignore metrics changes
	resourcePool.Metrics = ResourcePoolMetrics{}

	expected := &ResourcePool{
		ID:               "d21f84fd-5063-4383-b2b0-65b9f25eac27",
		Name:             "Resources",
		MachineManagerID: "9dba240e-a605-4103-bac7-5336d3ffd124",
		Moref:            "resgroup-1042",
		Parent: ResourcePoolParent{
			ID:   "domain-c1041",
			Type: "ClusterComputeResource",
		},
	}
	require.Equal(t, expected, resourcePool)
}
