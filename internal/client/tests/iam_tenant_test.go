package client

import (
	"context"
	"os"
	"testing"

	clientpkg "github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/stretchr/testify/require"
)

func TestIAM_Tenants(t *testing.T) {
	tenants, err := client.IAM().Tenant().List(context.Background())
	require.NoError(t, err)

	require.Len(t, tenants, 2)

	companyID := os.Getenv(testCompanyIDEnvName)
	expected := &clientpkg.Tenant{
		ID:        os.Getenv(TenantId),
		Name:      os.Getenv(TenantName),
		SNC:       true,
		CompanyID: companyID,
	}

	var rightTenant = &clientpkg.Tenant{}
	for _, tenant := range tenants {
		if tenant.ID == expected.ID {
			rightTenant = tenant
		}
	}

	require.Equal(t, expected, rightTenant)
}
