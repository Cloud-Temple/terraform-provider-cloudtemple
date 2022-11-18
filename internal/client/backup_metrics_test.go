package client

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBackupMetricsClient_History(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := client.Backup().Metrics().History(ctx, 4)
	require.NoError(t, err)
}

func TestBackupMetricsClient_Coverage(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := client.Backup().Metrics().Coverage(ctx)
	require.NoError(t, err)
}

func TestBackupMetricsClient_VirtualMachines(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := client.Backup().Metrics().VirtualMachines(ctx)
	require.NoError(t, err)
}

func TestBackupMetricsClient_Policies(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	policiesMetrics, err := client.Backup().Metrics().Policies(ctx)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(policiesMetrics), 1)

	expected := &BackupMetricsPolicies{
		Name:        "SLA_DAILY",
		TriggerType: "DAILY",
	}

	var found bool
	for _, pm := range policiesMetrics {
		if pm.Name == "SLA_DAILY" {
			// Ignore some fields
			pm.NumberOfProtectedVM = 0

			require.Equal(t, expected, pm)

			found = true
			break
		}
	}
	require.True(t, found)
}

func TestBackupMetricsClient_Platform(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	platformMetrics, err := client.Backup().Metrics().Platform(ctx)
	require.NoError(t, err)

	expected := &BackupMetricsPlatform{
		Version:        "10.1.12",
		Build:          "124",
		Date:           "Wed Sep  7 14:02:15 EDT 2022",
		Product:        "Spectrum Protect Plus",
		Epoch:          1662573735000,
		DeploymentType: "standard",
	}

	require.Equal(t, expected, platformMetrics)
}

func TestBackupMetricsClient_PlatformCPU(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	_, err := client.Backup().Metrics().PlatformCPU(ctx)
	require.NoError(t, err)
}
