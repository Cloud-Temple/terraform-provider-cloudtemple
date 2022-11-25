package client

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

const (
	testClientIDEnvName  = "SHIVA_TEST_CLIENT_ID"
	testSecretIDEnvName  = "SHIVA_TEST_SECRET_ID"
	testCompanyIDEnvName = "SHIVA_TEST_COMPANY_ID"
)

var client *Client = nil

func TestMain(m *testing.M) {
	envNames := []string{
		testClientIDEnvName,
		testSecretIDEnvName,
		testCompanyIDEnvName,
	}

	var fail bool
	for _, name := range envNames {
		if os.Getenv(name) == "" {
			fmt.Fprintf(os.Stderr, "%s must be set to run the tests\n", name)
			fail = true
		}
	}
	if fail {
		os.Exit(1)
	}

	config := DefaultConfig()
	config.ClientID = os.Getenv(testClientIDEnvName)
	config.SecretID = os.Getenv(testSecretIDEnvName)

	config.errorOnUnexpectedActivity = true

	c, err := NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	client = c

	// Clean resources from previous tests run
	ctx := context.Background()
	vms, err := client.Compute().VirtualMachine().List(ctx, true, "", false, false, nil, nil, nil, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	for _, vm := range vms {
		if strings.HasPrefix(vm.Name, "test-client") {
			if vm.PowerState == "running" {
				vm, err = client.Compute().VirtualMachine().Read(ctx, vm.ID)
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err.Error())
					os.Exit(1)
				}

				activityId, err := client.Compute().VirtualMachine().Power(ctx, &PowerRequest{
					ID:           vm.ID,
					DatacenterId: vm.VirtualDatacenterId,
					PowerAction:  "off",
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err.Error())
					os.Exit(1)
				}
				_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to stop %s:%s\n", vm.Name, err.Error())
					os.Exit(1)
				}
			}

			activityId, err := client.Compute().VirtualMachine().Delete(ctx, vm.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
			_, err = client.Activity().WaitForCompletion(ctx, activityId, nil)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete %s: %s\n", vm.Name, err.Error())
				os.Exit(1)
			}
		}
	}

	names := map[string]struct{}{
		"client-test": {},
	}

	lt, err := client.Token(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	tokens, err := client.IAM().PAT().List(ctx, lt.UserID(), lt.TenantID())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	for _, token := range tokens {
		if _, found := names[token.Name]; found {
			err := client.IAM().PAT().Delete(ctx, token.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
		}
	}

	os.Exit(m.Run())
}

func testTenantID(t *testing.T) string {
	t.Helper()

	tenants, err := client.IAM().Tenant().List(context.Background())
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tenants), 1)

	return tenants[0].ID
}

func testUserID(t *testing.T) string {
	t.Helper()

	lt, err := client.Token(context.Background())
	require.NoError(t, err)

	return lt.UserID()
}

func testRole(t *testing.T) *Role {
	t.Helper()

	roles, err := client.IAM().Role().List(context.Background())
	require.NoError(t, err)

	var role *Role
	for _, r := range roles {
		if r.Name == "iam_read" {
			role = r
			break
		}
	}
	require.NotNil(t, role)

	return role
}

func TestAPI_token(t *testing.T) {
	t.Parallel()

	token, err := client.token(context.Background())
	require.NoError(t, err)
	require.NotNil(t, token)
	require.NotNil(t, client.savedToken)
}

func TestAPI_tokenCache(t *testing.T) {
	t.Parallel()

	token, err := client.token(context.Background())
	require.NoError(t, err)

	newToken, err := client.token(context.Background())
	require.NoError(t, err)

	require.Equal(t, token.Raw, newToken.Raw)
}

func TestAPI_tokenExpiration(t *testing.T) {
	t.Parallel()

	if os.Getenv("CLIENT_RUN_LONG_TESTS") == "" {
		t.Skip("Set the CLIENT_RUN_LONG_TESTS environment variable to run this test")
	}

	token, err := client.token(context.Background())
	require.NoError(t, err)
	require.NotNil(t, token)

	time.Sleep(28 * time.Minute)
	newToken, err := client.token(context.Background())
	require.NoError(t, err)
	require.NotNil(t, token)

	require.NotEqual(t, token.Raw, newToken.Raw)
}
