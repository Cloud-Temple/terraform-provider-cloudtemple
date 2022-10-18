package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
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

// func getTestClient(t *testing.T) (*client.Client, string, string) {
// 	config := client.DefaultConfig()
// 	config.ClientID = os.Getenv("CLOUDTEMPLE_CLIENT_ID")
// 	config.SecretID = os.Getenv("CLOUDTEMPLE_SECRET_ID")

// 	client, err := client.NewClient(config)
// 	if err != nil {
// 		t.Fatalf("fail to get test client: %s", err)
// 	}

// 	lt, err := client.Token(context.Background())
// 	if err != nil {
// 		t.Fatalf("failed to get token: %s", err)
// 	}

// 	return client, lt.UserID(), lt.TenantID()
// }
