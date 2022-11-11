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

func TestUUIDValidation(t *testing.T) {
	provider := New("dev")()

	checkUUID := func(name string, r *schema.Resource) func(t *testing.T) {
		expected := reflect.ValueOf(validation.IsUUID).Pointer()

		var validateSchema func(r *schema.Resource)
		validateSchema = func(r *schema.Resource) {
			for n, s := range r.Schema {
				if !strings.HasSuffix(n, "id") {
					return
				}
				if !s.Optional && !s.Required {
					return
				}
				if reflect.ValueOf(s.ValidateFunc).Pointer() != expected {
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
		t.Run(name, checkUUID("data."+name, datasource))
	}

	for name, resource := range provider.ResourcesMap {
		t.Run(name, checkUUID(name, resource))
	}
}
