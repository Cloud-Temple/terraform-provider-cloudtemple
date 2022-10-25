package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceDatastore(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceDatastore,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore.foo", "id", "d439d467-943a-49f5-a022-c0c25b737022"),
					resource.TestCheckResourceAttr("data.cloudtemple_compute_datastore.foo", "name", "ds001-bob-svc1-data4-eqx6"),
				),
			},
		},
	})
}

const testAccDataSourceDatastore = `
data "cloudtemple_compute_datastore" "foo" {
  id = "d439d467-943a-49f5-a022-c0c25b737022"
}
`
