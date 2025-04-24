package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupJobSessionClient_List(t *testing.T) {
	ctx := context.Background()
	jobSessions, err := client.Backup().JobSession().List(ctx, nil)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(jobSessions), 1)
}
