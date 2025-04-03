package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	MetricPolicyName          = "BACKUP_METRICS_POLICY_NAME"
	MetricPolicyTriggerType   = "BACKUP_METRICS_POLICY_TRIGGER_TYPE"
	MetricsPlatformVersion    = "BACKUP_METRICS_PLATEFORM_VERSION"
	MetricsPlatformBuild      = "BACKUP_METRICS_PLATEFORM_BUILD"
	MetricsPlatformDate       = "BACKUP_METRICS_PLATEFORM_DATE"
	MetricsPlatformProduct    = "BACKUP_METRICS_PLATEFORM_PRODUCT"
	MetricsPlatformEpoch      = "BACKUP_METRICS_PLATEFORM_EPOCH"
	MetricsPlatformDeployType = "BACKUP_METRICS_PLATEFORM_DEPLOY_TYPE"
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

	expected := &clientpkg.BackupMetricsPolicies{
		Name:        os.Getenv(MetricPolicyName),
		TriggerType: os.Getenv(MetricPolicyTriggerType),
	}

	var found bool
	for _, pm := range policiesMetrics {
		if pm.Name == os.Getenv(MetricPolicyName) {
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
	ctx := context.Background()
	platformMetrics, err := client.Backup().Metrics().Platform(ctx)
	require.NoError(t, err)

	require.Equal(t, os.Getenv(MetricsPlatformVersion), platformMetrics.Version)
	require.Equal(t, os.Getenv(MetricsPlatformDeployType), platformMetrics.DeploymentType)
}

func TestBackupMetricsClient_PlatformCPU(t *testing.T) {
	ctx := context.Background()
	_, err := client.Backup().Metrics().PlatformCPU(ctx)
	require.NoError(t, err)
}
