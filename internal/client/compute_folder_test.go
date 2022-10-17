package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_FolderList(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	folders, err := client.Compute().Folder().List(ctx, "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(folders), 1)

	var found bool
	for _, f := range folders {
		if f.ID == "b41ea9b1-4cca-44ed-9a76-2b598de03781" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_FolderRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	folder, err := client.Compute().Folder().Read(ctx, "b41ea9b1-4cca-44ed-9a76-2b598de03781")
	require.NoError(t, err)

	expected := &Folder{
		ID:               "b41ea9b1-4cca-44ed-9a76-2b598de03781",
		Name:             "Datacenters",
		MachineManagerId: "9dba240e-a605-4103-bac7-5336d3ffd124",
	}
	require.Equal(t, expected, folder)
}
