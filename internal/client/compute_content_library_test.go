package client

import (
	"context"
	"testing"
	"time"

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

func TestContentLibraryClient_ListItems(t *testing.T) {
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

	// ignore some fields for the test
	clItem.LastModifiedTime = ""

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
			OvfProperties:    []string(nil),
		},
		clItem,
	)
}

func TestContentLibraryClient_ReadItem(t *testing.T) {
	ctx := context.Background()
	item, err := client.Compute().ContentLibrary().ReadItem(ctx, "355b654d-6ea2-4773-80ee-246d3f56964f", "8faded09-9f8b-4e27-a978-768f72f8e5f8")
	require.NoError(t, err)

	// ignore some fields for the test
	item.LastModifiedTime = ""

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
			OvfProperties:    []string{},
		},
		item,
	)
}

func TestContentLibraryClient_Clone(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().ContentLibrary().Deploy(ctx, &ComputeContentLibraryItemDeployRequest{
		ContentLibraryId:     "355b654d-6ea2-4773-80ee-246d3f56964f",
		ContentLibraryItemId: "8faded09-9f8b-4e27-a978-768f72f8e5f8",
		Name:                 "test-client-content-library-deploy",
		HostClusterId:        "dde72065-60f4-4577-836d-6ea074384d62",
		DatacenterId:         "85d53d08-0fa9-491e-ab89-90919516df25",
		DatastoreId:          "d439d467-943a-49f5-a022-c0c25b737022",
	})
	require.NoError(t, err)

	activity, err := client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)

	instanceId := activity.State["completed"].Result
	require.NotZero(t, instanceId)

	vm, err := client.Compute().VirtualMachine().Read(ctx, instanceId)
	require.NoError(t, err)
	require.Equal(t, "test-client-content-library-deploy", vm.Name)

	activityId, err = client.Compute().VirtualMachine().Delete(ctx, instanceId)
	require.NoError(t, err)
	_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
	require.NoError(t, err)
}
