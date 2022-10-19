package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCompute_ContentLibraryList(t *testing.T) {
	ctx := context.Background()
	contentLibraries, err := client.Compute().ContentLibrary().List(ctx, "", "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(contentLibraries), 1)

	var found bool
	for _, cl := range contentLibraries {
		if cl.ID == "355b654d-6ea2-4773-80ee-246d3f56964f" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_ContentLibraryRead(t *testing.T) {
	ctx := context.Background()
	contentLibrary, err := client.Compute().ContentLibrary().Read(ctx, "355b654d-6ea2-4773-80ee-246d3f56964f")
	require.NoError(t, err)

	expected := &ContentLibrary{
		ID:               "355b654d-6ea2-4773-80ee-246d3f56964f",
		Name:             "PUBLIC",
		MachineManagerID: "9dba240e-a605-4103-bac7-5336d3ffd124",
		Type:             "SUBSCRIBED",
		Datastore: DatastoreLink{
			ID:   "24371f16-b480-40d3-9587-82f97933abca",
			Name: "ds002-bob-svc1-stor4-th3",
		},
	}
	require.Equal(t, expected, contentLibrary)
}
