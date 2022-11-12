package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupSLAPolicyClient_List(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	slaPolicies, err := client.Backup().SLAPolicy().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(slaPolicies), 1)

	var found bool
	for _, sl := range slaPolicies {
		if sl.ID == "442718ef-44a1-43d7-9b57-2d910d74e928" {
			found = true
			break
		}
	}
	require.True(t, found)
}

func TestBackupSLAPolicyClient_Read(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	slaPolicy, err := client.Backup().SLAPolicy().Read(ctx, "442718ef-44a1-43d7-9b57-2d910d74e928")
	require.NoError(t, err)

	expected := &BackupSLAPolicy{
		ID:   "442718ef-44a1-43d7-9b57-2d910d74e928",
		Name: "SLA_ADMIN",
		SubPolicies: []*BackupSLASubPolicy{
			{
				Type:          "REPLICATION",
				UseEncryption: false,
				Software:      true,
				Site:          "DC-EQX6",
				Retention: BackupSLAPolicyRetention{
					Age: 15,
				},
				Trigger: BackupSLAPolicyTrigger{
					Frequency:    1,
					Type:         "DAILY",
					ActivateDate: 1568617200000,
				},
			},
		},
	}

	require.Equal(t, expected, slaPolicy)
}
