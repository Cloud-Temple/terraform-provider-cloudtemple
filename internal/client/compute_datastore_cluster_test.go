package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_DatastoreClusterList(t *testing.T) {
	ctx := context.Background()
	datastoreClusters, err := client.Compute().DatastoreCluster().List(ctx, "", "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(datastoreClusters), 1)

	var found bool
	for _, dc := range datastoreClusters {
		if dc.ID == "6b06b226-ef55-4a0a-92bc-7aa071681b1b" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_DatastoreClusterRead(t *testing.T) {
	ctx := context.Background()
	datastoreCluster, err := client.Compute().DatastoreCluster().Read(ctx, "6b06b226-ef55-4a0a-92bc-7aa071681b1b")
	require.NoError(t, err)

	expected := &DatastoreCluster{
		ID:               "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		Name:             "sdrs001-LIVE_KOUKOU",
		Moref:            "group-p1055",
		MachineManagerId: "9dba240e-a605-4103-bac7-5336d3ffd124",
		Datastores:       []string{"d439d467-943a-49f5-a022-c0c25b737022"},
		Metrics: DatastoreClusterMetrics{
			FreeCapacity:                  131042639872,
			MaxCapacity:                   536602476544,
			Enabled:                       true,
			DefaultVmBehavior:             "manual",
			LoadBalanceInterval:           480,
			SpaceThresholdMode:            "utilization",
			SpaceUtilizationThreshold:     80,
			MinSpaceUtilizationDifference: 0,
			ReservablePercentThreshold:    0,
			ReservableThresholdMode:       "",
			IoLatencyThreshold:            0,
			IoLoadImbalanceThreshold:      0,
			IoLoadBalanceEnabled:          false,
		},
	}
	require.Equal(t, expected, datastoreCluster)
}
