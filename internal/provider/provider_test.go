package provider

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

const (
	testClientIDEnvName  = "SHIVA_TEST_CLIENT_ID"
	testSecretIDEnvName  = "SHIVA_TEST_SECRET_ID"
	testCompanyIDEnvName = "SHIVA_TEST_COMPANY_ID"
)

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

	config := client.DefaultConfig()
	config.ClientID = os.Getenv(testClientIDEnvName)
	config.SecretID = os.Getenv(testSecretIDEnvName)

	c, err := client.NewClient(config)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}

	// Clean resources from previous tests run
	ctx := context.Background()
	vms, err := c.Compute().VirtualMachine().List(ctx, true, "", false, false, nil, nil, nil, nil, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	for _, vm := range vms {
		if strings.HasPrefix(vm.Name, "test-terraform") {
			vm, err = c.Compute().VirtualMachine().Read(ctx, vm.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}

			if vm.PowerState == "running" {
				activityId, err := c.Compute().VirtualMachine().Power(ctx, &client.PowerRequest{
					ID:           vm.ID,
					DatacenterId: vm.VirtualDatacenterId,
					PowerAction:  "off",
				})
				if err != nil {
					fmt.Fprintf(os.Stderr, "%s\n", err.Error())
					os.Exit(1)
				}
				_, err = c.Activity().WaitForCompletion(ctx, activityId)
				if err != nil {
					fmt.Fprintf(os.Stderr, "failed to stop %s:%s\n", vm.Name, err.Error())
					os.Exit(1)
				}
			}

			activityId, err := c.Compute().VirtualMachine().Delete(ctx, vm.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
			_, err = c.Activity().WaitForCompletion(ctx, activityId)
			if err != nil {
				fmt.Fprintf(os.Stderr, "failed to delete %s: %s\n", vm.Name, err.Error())
				os.Exit(1)
			}
		}
	}

	names := map[string]struct{}{
		"client-test": {},
	}

	lt, err := c.Token(context.Background())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	tokens, err := c.IAM().PAT().List(ctx, lt.UserID(), lt.TenantID())
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err.Error())
		os.Exit(1)
	}
	for _, token := range tokens {
		if _, found := names[token.Name]; found {
			err := c.IAM().PAT().Delete(ctx, token.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "%s\n", err.Error())
				os.Exit(1)
			}
		}
	}

	os.Exit(m.Run())
}

// providerFactories are used to instantiate a provider during acceptance testing.
// The factory function will be invoked for every Terraform CLI command executed
// to create a provider server to which the CLI can reattach.
var providerFactories = map[string]func() (*schema.Provider, error){
	"cloudtemple": func() (*schema.Provider, error) {
		return New("dev")(), nil
	},
}

func TestProvider(t *testing.T) {
	if err := New("dev")().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func testAccPreCheck(t *testing.T) {}

func TestIDValidation(t *testing.T) {
	provider := New("dev")()

	checkUUID := func(name string, r *schema.Resource) func(t *testing.T) {
		expected := map[uintptr]struct{}{
			reflect.ValueOf(validation.IsUUID).Pointer(): {},
			reflect.ValueOf(IsNumber).Pointer():          {},
		}

		var validateSchema func(r *schema.Resource)
		validateSchema = func(r *schema.Resource) {
			for n, s := range r.Schema {
				if !strings.HasSuffix(n, "id") {
					return
				}
				if !s.Optional && !s.Required {
					return
				}
				if _, found := expected[reflect.ValueOf(s.ValidateFunc).Pointer()]; !found {
					t.Errorf("%s.%s ValidateFunc is incorrect", name, n)
				}
				if resource, ok := s.Elem.(*schema.Resource); ok {
					validateSchema(resource)
				}
			}
		}

		return func(t *testing.T) {
			validateSchema(r)
		}
	}

	for name, datasource := range provider.DataSourcesMap {
		t.Run("data."+name, checkUUID("data."+name, datasource))
	}

	for name, resource := range provider.ResourcesMap {
		t.Run(name, checkUUID(name, resource))
	}
}

func TestImport(t *testing.T) {
	provider := New("dev")()

	skip := map[string]struct{}{
		// Access tokens cannot be imported because there is no way of getting the secret
		"cloudtemple_iam_personal_access_token": {},

		// TODO: we skip this one for now
		"cloudtemple_compute_network_adapter": {},
	}

	for name, resource := range provider.ResourcesMap {
		t.Run(name, func(t *testing.T) {
			if _, found := skip[name]; found {
				t.Skip()
			}
			if resource.Importer == nil {
				t.Fatalf("no importer for %s", name)
			}
		})
	}
}
