package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	ContentLibraryId       = "COMPUTE_CONTENT_LIBRARY_ID"
	ContentLibraryName     = "COMPUTE_CONTENT_LIBRARY_NAME"
	ContentLibraryType     = "COMPUTE_CONTENT_LIBRARY_TYPE"
	ContentLibraryItemId   = "COMPUTE_CONTENT_LIBRARY_ITEM_ID"
	ContentLibraryItemName = "COMPUTE_CONTENT_LIBRARY_ITEM_NAME"
	ContentLibraryItemType = "COMPUTE_CONTENT_LIBRARY_ITEM_TYPE"
	MachineManagerId       = "COMPUTE_VCENTER_ID"
	DataStoreId            = "COMPUTE_DATASTORE_ID"
	DataStoreName          = "COMPUTE_DATASTORE_NAME"
)

func TestCompute_ContentLibraryList(t *testing.T) {
	ctx := context.Background()
	contentLibraries, err := client.Compute().ContentLibrary().List(ctx, &clientpkg.ContentLibraryFilter{
		Name:             os.Getenv(ContentLibraryName),
		MachineManagerId: os.Getenv(MachineManagerId),
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(contentLibraries), 1)

	var found bool
	for _, cl := range contentLibraries {
		if cl.ID == os.Getenv(ContentLibraryId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_ContentLibraryRead(t *testing.T) {
	ctx := context.Background()
	contentLibrary, err := client.Compute().ContentLibrary().Read(ctx, os.Getenv(ContentLibraryId))
	require.NoError(t, err)

	expected := &clientpkg.ContentLibrary{
		ID:   os.Getenv(ContentLibraryId),
		Name: os.Getenv(ContentLibraryName),
		Type: os.Getenv(ContentLibraryType),
		Datastore: clientpkg.DatastoreLink{
			ID:   os.Getenv(DataStoreId),
			Name: os.Getenv(DataStoreName),
		},
	}
	require.Equal(t, expected, contentLibrary)
}

func TestContentLibraryClient_ListItems(t *testing.T) {
	ctx := context.Background()
	items, err := client.Compute().ContentLibrary().ListItems(ctx, &clientpkg.ContentLibraryItemFilter{
		ContentLibraryId: os.Getenv(ContentLibraryId),
	})
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(items), 1)

	var clItem *clientpkg.ContentLibraryItem
	for _, item := range items {
		if item.ID == os.Getenv(ContentLibraryItemId) {
			clItem = item
			break
		}
	}
	require.NotNil(t, clItem)

	require.Equal(t, os.Getenv(ContentLibraryItemId), clItem.ID)
	require.Equal(t, os.Getenv(ContentLibraryItemName), clItem.Name)
}

func TestContentLibraryClient_ReadItem(t *testing.T) {
	ctx := context.Background()
	item, err := client.Compute().ContentLibrary().ReadItem(ctx, os.Getenv(ContentLibraryId), os.Getenv(ContentLibraryItemId))
	require.NoError(t, err)

	require.Equal(t, os.Getenv(ContentLibraryItemId), item.ID)
	require.Equal(t, os.Getenv(ContentLibraryItemName), item.Name)
	require.Equal(t, os.Getenv(ContentLibraryId), item.ContentLibraryId)
	require.Equal(t, os.Getenv(ContentLibraryItemType), item.Type)
}

func TestContentLibraryClient_Clone(t *testing.T) {
	ctx := context.Background()
	activityId, err := client.Compute().ContentLibrary().Deploy(ctx, &clientpkg.ComputeContentLibraryItemDeployRequest{
		ContentLibraryId:     os.Getenv(ContentLibraryId),
		ContentLibraryItemId: os.Getenv(ContentLibraryItemId),
		Name:                 "test-client-content-library-deploy",
		HostClusterId:        os.Getenv(HostClusterId),
		DatacenterId:         os.Getenv(DataCenterId),
		DatastoreId:          os.Getenv(DataStoreId),
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
