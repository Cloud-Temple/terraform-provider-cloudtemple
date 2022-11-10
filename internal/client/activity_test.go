package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestActivity_List(t *testing.T) {
	ctx := context.Background()
	activities, err := client.Activity().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(activities), 1)

	var found bool
	for _, a := range activities {
		if a.ID == "022ae273-552d-4588-a913-f8260638d3a4" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestActivity_Read(t *testing.T) {
	ctx := context.Background()
	activity, err := client.Activity().Read(ctx, "022ae273-552d-4588-a913-f8260638d3a4")
	require.NoError(t, err)

	expected := &Activity{
		ID:           "022ae273-552d-4588-a913-f8260638d3a4",
		TenantId:     "e225dbf8-e7c5-4664-a595-08edf3526080",
		Description:  "Updating virtual machine test-terraform",
		Type:         "ComputeActivity",
		Tags:         []string{"compute", "vcenter", "virtual_machine", "update"},
		CreationDate: time.Date(2022, time.November, 9, 15, 34, 34, 659000000, time.UTC),
		State: map[string]ActivityState{
			"completed": {
				StartDate: "2022-11-09T15:34:34.660Z",
				StopDate:  "2022-11-09T15:34:34.720Z",
			},
		},
		ConcernedItems: []ActivityConcernedItem{
			{
				ID:   "6453cd41-1d08-4caf-935f-99c48be4a994",
				Type: "virtual_machine",
			},
		},
	}

	require.Equal(t, expected, activity)
}
