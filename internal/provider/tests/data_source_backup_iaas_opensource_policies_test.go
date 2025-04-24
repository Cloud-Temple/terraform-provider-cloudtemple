package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Utilise la constante définie dans data_source_backup_openiaas_policy_test.go
// OpenIaaSMachineManagerId = "COMPUTE_IAAS_OPENSOURCE_AVAILABILITY_ZONE_ID"

func TestAccDataSourceOpenIaaSBackupPolicies(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSBackupPolicies,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des policies n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_iaas_opensource_policies.foo",
						"policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse policies count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected policies list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.foo", "policies.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.foo", "policies.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.foo", "policies.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.foo", "policies.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.foo", "policies.0.machine_manager_name"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSBackupPoliciesWithFilter, os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des policies n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_backup_iaas_opensource_policies.filtered",
						"policies.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse policies count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected policies list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.filtered", "policies.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.filtered", "policies.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.filtered", "policies.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.filtered", "policies.0.machine_manager_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_policies.filtered", "policies.0.machine_manager_name"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSBackupPolicies = `
data "cloudtemple_backup_iaas_opensource_policies" "foo" {}
`

const testAccDataSourceOpenIaaSBackupPoliciesWithFilter = `
data "cloudtemple_backup_iaas_opensource_policies" "filtered" {
  machine_manager_id = "%s"
}
`
