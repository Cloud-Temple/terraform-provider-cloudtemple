package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_HostList(t *testing.T) {
	ctx := context.Background()
	hosts, err := client.Compute().Host().List(ctx, "", "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(hosts), 1)

	var found bool
	for _, h := range hosts {
		if h.ID == "8997db63-24d5-47f4-8cca-d5f5df199d1a" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_HostRead(t *testing.T) {
	ctx := context.Background()
	host, err := client.Compute().Host().Read(ctx, "8997db63-24d5-47f4-8cca-d5f5df199d1a")
	require.NoError(t, err)

	// ignore metrics changes
	host.Metrics = HostMetrics{}
	host.VirtualMachines = nil

	expected := &Host{
		ID:               "8997db63-24d5-47f4-8cca-d5f5df199d1a",
		Name:             "esx001-bob-ucs01-eqx6.cloud-temple.lan",
		Moref:            "host-1046",
		MachineManagerID: "",
		Metrics:          HostMetrics{},
	}
	require.Equal(t, expected, host)
}
