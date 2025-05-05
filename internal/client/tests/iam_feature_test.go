package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

const (
	FeatureId   = "IAM_FEATURE_ID"
	FeatureName = "IAM_FEATURE_NAME"
)

func TestIAM_Features(t *testing.T) {
	features, err := client.IAM().Feature().List(context.Background())
	require.NoError(t, err)

	var rtms *clientpkg.Feature
	for _, feature := range features {
		if feature.Name == "rtms" {
			rtms = feature
			break
		}
	}
	require.NotNil(t, rtms, "rtms has not been found")
	require.Equal(t, os.Getenv(FeatureName), rtms.Name)
	require.Equal(t, os.Getenv(FeatureId), rtms.ID)
	require.Len(t, rtms.SubFeatures, 2)
}

func TestIAM_FeatureAssignments(t *testing.T) {
	ctx := context.Background()
	tenants, err := client.IAM().Tenant().List(ctx)
	require.NoError(t, err)
	require.Len(t, tenants, 2)

	tenant := tenants[0]
	fa, err := client.IAM().Feature().ListAssignments(ctx, tenant.ID)
	require.NoError(t, err)

	require.GreaterOrEqual(t, len(fa), 1)
	require.NotEqual(t, "", fa[0].FeatureID)
	require.NotEqual(t, "", fa[0].TenantID)
}
