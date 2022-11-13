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
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "tenant_id", "e225dbf8-e7c5-4664-a595-08edf3526080"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "description", "Creating virtual machine test-power."),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "type", "ComputeActivity"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "creation_date", "2022-11-12T22:54:53Z"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "tags.#", "4"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.#", "1"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.0.id", "019751ae-15c2-468f-98a6-d7cbd90a83d0"),
					resource.TestCheckResourceAttr("data.cloudtemple_activity.foo", "concerned_items.0.type", "virtual_machine"),
				),
			},
		},
	})
}

const testAccDataActivity = `
data "cloudtemple_activity" "foo" {
  id = "00791ba3-8cc0-4051-a654-9cd4d71eb48c"
}
`
