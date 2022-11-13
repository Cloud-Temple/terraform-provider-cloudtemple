package client

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestActivity_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	activities, err := client.Activity().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(activities), 1)

	var found bool
	for _, a := range activities {
		if a.ID == "00791ba3-8cc0-4051-a654-9cd4d71eb48c" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestActivity_Read(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	activity, err := client.Activity().Read(ctx, "00791ba3-8cc0-4051-a654-9cd4d71eb48c")
	require.NoError(t, err)

	expected := &Activity{
		ID:           "00791ba3-8cc0-4051-a654-9cd4d71eb48c",
		TenantId:     "e225dbf8-e7c5-4664-a595-08edf3526080",
		Description:  "Creating virtual machine test-power.",
		Type:         "ComputeActivity",
		Tags:         []string{"compute", "vcenter", "virtual_machine", "create"},
		CreationDate: time.Date(2022, time.November, 12, 22, 54, 53, 72000000, time.UTC),
		State: map[string]ActivityState{
			"completed": {
				StartDate: "2022-11-12T22:54:53.073Z",
				StopDate:  "2022-11-12T22:54:57.379Z",
			},
		},
		ConcernedItems: []ActivityConcernedItem{
			{
				ID:   "019751ae-15c2-468f-98a6-d7cbd90a83d0",
				Type: "virtual_machine",
			},
		},
	}

	require.Equal(t, expected, activity)
}

func TestActivityClient_WaitForCompletion(t *testing.T) {
	ctx := context.Background()

	tests := map[string]struct {
		id      string
		want    *Activity
		wantErr string
	}{
		"finished activity": {
			id: "00791ba3-8cc0-4051-a654-9cd4d71eb48c",
			want: &Activity{
				ID:           "00791ba3-8cc0-4051-a654-9cd4d71eb48c",
				TenantId:     "e225dbf8-e7c5-4664-a595-08edf3526080",
				Description:  "Creating virtual machine test-power.",
				Type:         "ComputeActivity",
				Tags:         []string{"compute", "vcenter", "virtual_machine", "create"},
				CreationDate: time.Date(2022, time.November, 12, 22, 54, 53, 72000000, time.UTC),
				State: map[string]ActivityState{
					"completed": {
						StartDate: "2022-11-12T22:54:53.073Z",
						StopDate:  "2022-11-12T22:54:57.379Z",
					},
				},
				ConcernedItems: []ActivityConcernedItem{
					{
						ID:   "019751ae-15c2-468f-98a6-d7cbd90a83d0",
						Type: "virtual_machine",
					},
				},
			},
		},
		"non-exisiting activity": {
			id:      "12345678-1234-5678-1234-567812345678",
			wantErr: `the activity "12345678-1234-5678-1234-567812345678" could not be found`,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := client.Activity().WaitForCompletion(ctx, tt.id)
			if tt.wantErr == "" {
				require.NoError(t, err)
			} else {
				require.ErrorContains(t, err, tt.wantErr)
			}
			require.Equal(t, tt.want, got)
		})
	}
}
