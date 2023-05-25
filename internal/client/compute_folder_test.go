package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	FolderId   = "TEST_COMPUTE_FOLDER_ID"
	FolderName = "TEST_COMPUTE_FOLDER_NAME"
)

func TestCompute_FolderList(t *testing.T) {
	ctx := context.Background()
	folders, err := client.Compute().Folder().List(ctx, "", "")
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(folders), 1)

	var found bool
	for _, f := range folders {
		if f.ID == os.Getenv(FolderId) {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestCompute_FolderRead(t *testing.T) {
	ctx := context.Background()
	folder, err := client.Compute().Folder().Read(ctx, os.Getenv(FolderId))
	require.NoError(t, err)

	expected := &Folder{
		ID:               os.Getenv(FolderId),
		Name:             os.Getenv(FolderName),
		MachineManagerId: os.Getenv(MachineManagerId),
	}
	require.Equal(t, expected, folder)
}
