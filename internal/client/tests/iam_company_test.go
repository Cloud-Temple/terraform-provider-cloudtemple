package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestIAM_Company(t *testing.T) {
	ctx := context.Background()
	companyID := os.Getenv(testCompanyIDEnvName)
	company, err := client.IAM().Company().Read(ctx, companyID)
	require.NoError(t, err)

	expected := &clientpkg.Company{
		ID:   companyID,
		Name: "Cloud Temple",
	}
	require.Equal(t, expected, company)
}
