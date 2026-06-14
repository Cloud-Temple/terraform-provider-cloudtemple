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

	// The sub-features of rtms are leaves: the real IAM feature tree depth is 2.
	// This locks the depth assumption the datasource schema/flatten rely on
	// (see internal/provider/helpers/helper_iam_feature.go). If the platform
	// ever nests sub-features deeper, this fails loudly so the schema depth is
	// revisited before a deeper tree silently gets truncated.
	for _, sf := range rtms.SubFeatures {
		require.NotNil(t, sf)
		require.Empty(t, sf.SubFeatures, "sub-feature %q is expected to be a leaf (observed IAM feature tree depth is 2)", sf.Name)
	}
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
