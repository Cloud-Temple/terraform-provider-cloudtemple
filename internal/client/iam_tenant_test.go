package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_Tenants(t *testing.T) {
	tenants, err := client.IAM().Tenant().List(context.Background())
	require.NoError(t, err)

	require.Len(t, tenants, 2)

	companyID := os.Getenv(testCompanyIDEnvName)
	expected := &Tenant{
		ID:        os.Getenv(TenantId),
		Name:      os.Getenv(TenantName),
		SNC:       true,
		CompanyID: companyID,
	}

	var rightTenant = &Tenant{}
	for _, tenant := range tenants {
		if tenant.ID == expected.ID {
			rightTenant = tenant
		}
	}

	require.Equal(t, expected, rightTenant)
}
