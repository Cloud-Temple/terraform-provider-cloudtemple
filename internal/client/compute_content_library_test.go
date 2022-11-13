package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCompute_ContentLibraryList(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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

func TestContentLibraryClient_ListItems(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	items, err := client.Compute().ContentLibrary().ListItems(ctx, "355b654d-6ea2-4773-80ee-246d3f56964f")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(items), 1)

	var clItem *ContentLibraryItem
	for _, item := range items {
		if item.ID == "8faded09-9f8b-4e27-a978-768f72f8e5f8" {
			clItem = item
			break
		}
	}
	require.NotNil(t, clItem)

	require.Equal(
		t,
		&ContentLibraryItem{
			ID:               "8faded09-9f8b-4e27-a978-768f72f8e5f8",
			ContentLibraryId: "",
			Name:             "20211115132417_master_linux-centos-8",
			Description:      "Centos 8",
			Type:             "ovf",
			CreationTime:     time.Date(1, time.January, 1, 0, 0, 0, 0, time.UTC),
			Size:             0,
			Stored:           true,
			LastModifiedTime: "2022-11-13T22:00:09.540Z",
			OvfProperties:    []string(nil),
		},
		clItem,
	)
}

func TestContentLibraryClient_ReadItem(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	item, err := client.Compute().ContentLibrary().ReadItem(ctx, "355b654d-6ea2-4773-80ee-246d3f56964f", "8faded09-9f8b-4e27-a978-768f72f8e5f8")
	require.NoError(t, err)

	require.Equal(
		t,
		&ContentLibraryItem{
			ID:               "8faded09-9f8b-4e27-a978-768f72f8e5f8",
			ContentLibraryId: "355b654d-6ea2-4773-80ee-246d3f56964f",
			Name:             "20211115132417_master_linux-centos-8",
			Description:      "Centos 8",
			Type:             "ovf",
			CreationTime:     time.Date(2021, time.December, 2, 3, 26, 39, 156000000, time.UTC),
			Size:             1706045044,
			Stored:           true,
			LastModifiedTime: "2022-11-13T22:00:09.540Z",
			OvfProperties:    []string{},
		},
		item,
	)
}
