package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_Company(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	companyID := os.Getenv(testCompanyIDEnvName)
	company, err := client.IAM().Company().Read(ctx, companyID)
	require.NoError(t, err)

	expected := &Company{
		ID:   companyID,
		Name: "Cloud Temple",
	}
	require.Equal(t, expected, company)
}
