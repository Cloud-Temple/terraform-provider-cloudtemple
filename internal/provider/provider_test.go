package provider

import (
	"reflect"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

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
	}

	for name, resource := range provider.ResourcesMap {
		t.Run(name, func(t *testing.T) {
			if _, found := skip[name]; found {
				return
			}
			if resource.Importer == nil {
				t.Fatalf("no importer for %s", name)
			}
		})
	}
}
