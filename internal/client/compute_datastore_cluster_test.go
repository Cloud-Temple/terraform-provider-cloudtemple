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

	// Skip checking changes on metrics
	datastoreCluster.Metrics = DatastoreClusterMetrics{}

	expected := &DatastoreCluster{
		ID:               "6b06b226-ef55-4a0a-92bc-7aa071681b1b",
		Name:             "sdrs001-LIVE_KOUKOU",
		Moref:            "group-p1055",
		MachineManagerId: "9dba240e-a605-4103-bac7-5336d3ffd124",
		Datastores:       []string{"d439d467-943a-49f5-a022-c0c25b737022"},
		Metrics:          DatastoreClusterMetrics{},
	}
	require.Equal(t, expected, datastoreCluster)
}
