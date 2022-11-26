package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_HostClusterList(t *testing.T) {
	ctx := context.Background()
	hostClusters, err := client.Compute().HostCluster().List(ctx, "9dba240e-a605-4103-bac7-5336d3ffd124", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hostClusters), 1)

	var found bool
	for _, hc := range hostClusters {
		if hc.ID == "dde72065-60f4-4577-836d-6ea074384d62" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_HostClusterRead(t *testing.T) {
	ctx := context.Background()
	hostCluster, err := client.Compute().HostCluster().Read(ctx, "dde72065-60f4-4577-836d-6ea074384d62")
	require.NoError(t, err)

	// ignore changes to metrics
	hostCluster.Metrics = HostClusterMetrics{}
	hostCluster.VirtualMachinesNumber = 0

	expected := &HostCluster{
		ID:    "dde72065-60f4-4577-836d-6ea074384d62",
		Name:  "clu002-ucs01_FLO",
		Moref: "domain-c1041",
		Hosts: []HostClusterHostStub{
			{
				ID:   "host-1046",
				Type: "HostSystem",
			},
		},
		Metrics:          HostClusterMetrics{},
		MachineManagerId: "",
	}
	require.Equal(t, expected, hostCluster)
}
