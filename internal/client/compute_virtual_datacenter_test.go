package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_VirtualDatacenterList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualDatacenters, err := client.Compute().VirtualDatacenter().List(ctx, "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(virtualDatacenters), 1)

	var found bool
	for _, vd := range virtualDatacenters {
		if vd.ID == "ac33c033-693b-4fc5-9196-26df77291dbb" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_VirtualDatacenterRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	virtualDatacenter, err := client.Compute().VirtualDatacenter().Read(ctx, "ac33c033-693b-4fc5-9196-26df77291dbb")
	require.NoError(t, err)

	expected := &VirtualDatacenter{
		ID:               "ac33c033-693b-4fc5-9196-26df77291dbb",
		Name:             "DC-TH3",
		MachineManagerID: "9dba240e-a605-4103-bac7-5336d3ffd124",
		TenantID:         "e225dbf8-e7c5-4664-a595-08edf3526080",
	}
	require.Equal(t, expected, virtualDatacenter)
}
