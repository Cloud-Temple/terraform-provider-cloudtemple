package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_NetworkList(t *testing.T) {
	ctx := context.Background()
	networks, err := client.Compute().Network().List(ctx, "", "", "", "", "", "", "", "", true)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(networks), 1)

	var found bool
	for _, h := range networks {
		if h.ID == "5e029210-b433-4c45-93be-092cef684edc" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_NetworkRead(t *testing.T) {
	ctx := context.Background()
	network, err := client.Compute().Network().Read(ctx, "5e029210-b433-4c45-93be-092cef684edc")
	require.NoError(t, err)

	expected := &Network{
		ID:    "5e029210-b433-4c45-93be-092cef684edc",
		Name:  "VLAN_201",
		Moref: "dvportgroup-1054",
	}
	require.Equal(t, expected, network)
}
