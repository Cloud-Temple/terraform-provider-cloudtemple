package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupMetricsClient_History(t *testing.T) {
	ctx := context.Background()
	_, err := client.Backup().Metrics().History(ctx, 4)
	require.NoError(t, err)
}

func TestBackupMetricsClient_Coverage(t *testing.T) {
	ctx := context.Background()
	_, err := client.Backup().Metrics().Coverage(ctx)
	require.NoError(t, err)
}

func TestBackupMetricsClient_VirtualMachines(t *testing.T) {
	ctx := context.Background()
	_, err := client.Backup().Metrics().VirtualMachines(ctx)
	require.NoError(t, err)
}

func TestBackupMetricsClient_Policies(t *testing.T) {
	ctx := context.Background()
	policiesMetrics, err := client.Backup().Metrics().Policies(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(policiesMetrics), 1)
}

func TestBackupMetricsClient_Platform(t *testing.T) {
	ctx := context.Background()
	platformMetrics, err := client.Backup().Metrics().Platform(ctx)
	require.NoError(t, err)

	require.NotEmpty(t, platformMetrics.Version)
	require.NotEmpty(t, platformMetrics.DeploymentType)
}

func TestBackupMetricsClient_PlatformCPU(t *testing.T) {
	ctx := context.Background()
	_, err := client.Backup().Metrics().PlatformCPU(ctx)
	require.NoError(t, err)
}
