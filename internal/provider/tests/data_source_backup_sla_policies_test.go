package provider

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceBackupSLAPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceBackupSLAPolicies,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des politiques SLA n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_sla_policies.foo",
						"sla_policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse sla_policies count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected sla_policies list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policies.foo", "sla_policies.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policies.foo", "sla_policies.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policies.foo", "sla_policies.0.sub_policies.#"),

					// Vérifier que la liste des sous-politiques n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_sla_policies.foo",
						"sla_policies.0.sub_policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse sub_policies count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected sub_policies list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales de la première sous-politique
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policies.foo", "sla_policies.2.sub_policies.0.type"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_sla_policies.foo", "sla_policies.2.sub_policies.0.site"),
				),
			},
		},
	})
}

const testAccDataSourceBackupSLAPolicies = `
data "cloudtemple_backup_sla_policies" "foo" {}
`
