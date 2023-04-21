package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_HostClusterList(t *testing.T) {
	ctx := context.Background()
	hostClusters, err := client.Compute().HostCluster().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hostClusters), 1)

	var found bool
	for _, hc := range hostClusters {
		if hc.ID == "bd5d8bf4-953a-46fb-9997-45467ba1ae6f" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_HostClusterRead(t *testing.T) {
	ctx := context.Background()
	hostCluster, err := client.Compute().HostCluster().Read(ctx, "bd5d8bf4-953a-46fb-9997-45467ba1ae6f")
	require.NoError(t, err)

	// ignore changes to metrics
	hostCluster.Metrics = HostClusterMetrics{}
	hostCluster.VirtualMachinesNumber = 0

	expected := &HostCluster{
		ID:    "bd5d8bf4-953a-46fb-9997-45467ba1ae6f",
		Name:  "clu001-ucs12",
		Moref: "domain-c1008",
		Hosts: []HostClusterHostStub{
			{
				ID:   "host-1022",
				Type: "HostSystem",
			},
			{
				ID:   "host-1015",
				Type: "HostSystem",
			},
		},
		Metrics: HostClusterMetrics{},
	}
	require.Equal(t, expected, hostCluster)
}
