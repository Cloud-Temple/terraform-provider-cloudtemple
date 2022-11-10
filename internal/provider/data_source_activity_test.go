package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataActivity(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataActivity,
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "tenant_id", "e225dbf8-e7c5-4664-a595-08edf3526080"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "description", "Updating virtual machine test-terraform"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "type", "ComputeActivity"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "creation_date", "2022-11-09T15:34:34Z"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "tags.#", "4"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.0.id", "6453cd41-1d08-4caf-935f-99c48be4a994"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.0.type", "virtual_machine"),
				),
			},
		},
	})
}

const testAccDataActivity = `
data "cloudtemple_activity" "foo" {
  id = "022ae273-552d-4588-a913-f8260638d3a4"
}
`
