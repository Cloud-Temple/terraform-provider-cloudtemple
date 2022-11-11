package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_NetworkAdapterList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	networkAdapters, err := client.Compute().NetworkAdapter().List(ctx, "de2b8b80-8b90-414a-bc33-e12f61a4c05c")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networkAdapters), 1)

	var found bool
	for _, na := range networkAdapters {
		if na.ID == "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_NetworkAdapterRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	networkAdapter, err := client.Compute().NetworkAdapter().Read(ctx, "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86")
	require.NoError(t, err)

	expected := &NetworkAdapter{
		ID:               "c74060bf-ebb3-455a-b0b0-d0dcb79f3d86",
		VirtualMachineId: "de2b8b80-8b90-414a-bc33-e12f61a4c05c",
		Name:             "Network adapter 1",
		Type:             "VMXNET3",
		MacType:          "ASSIGNED",
		MacAddress:       "00:50:56:85:44:2e",
		Connected:        false,
		AutoConnect:      true,
	}
	require.Equal(t, expected, networkAdapter)
}
