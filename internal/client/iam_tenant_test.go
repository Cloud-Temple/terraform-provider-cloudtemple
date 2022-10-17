package client

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIAM_Tenants(t *testing.T) {
	companyID := os.Getenv(testCompanyIDEnvName)
	tenants, err := client.IAM().Tenant().List(context.Background(), companyID)
	require.NoError(t, err)

	require.Len(t, tenants, 1)

	expected := &Tenant{
		ID:        "e225dbf8-e7c5-4664-a595-08edf3526080",
		Name:      "BOB",
		SNC:       false,
		CompanyID: companyID,
	}
	require.Equal(t, expected, tenants[0])
}
