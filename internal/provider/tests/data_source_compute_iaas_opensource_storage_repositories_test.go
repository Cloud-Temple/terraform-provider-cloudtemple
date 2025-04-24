package provider

import (
	"fmt"
	"os"
	"strconv"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccDataSourceOpenIaaSStorageRepositories(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccDataSourceOpenIaaSStorageRepositories,
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des storage_repositories n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_storage_repositories.foo",
						"storage_repositories.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse storage_repositories count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected storage_repositories list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.foo", "storage_repositories.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.foo", "storage_repositories.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.foo", "storage_repositories.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.foo", "storage_repositories.0.machine_manager_id"),
				),
			},
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSStorageRepositoriesWithFilter, os.Getenv(OpenIaaSMachineManagerId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Vérifier que la liste des storage_repositories n'est pas vide
					resource.TestCheckResourceAttrWith(
						"data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered",
						"storage_repositories.#",
						func(value string) error {
							count, err := strconv.Atoi(value)
							if err != nil {
								return fmt.Errorf("failed to parse storage_repositories count: %s", err)
							}
							if count <= 0 {
								return fmt.Errorf("expected storage_repositories list to be non-empty, got %d items", count)
							}
							return nil
						},
					),
					// Vérifier les propriétés principales du premier élément
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered", "storage_repositories.0.id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered", "storage_repositories.0.name"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered", "storage_repositories.0.internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_compute_iaas_opensource_storage_repositories.filtered", "storage_repositories.0.machine_manager_id"),
				),
			},
		},
	})
}

const testAccDataSourceOpenIaaSStorageRepositories = `
data "cloudtemple_compute_iaas_opensource_storage_repositories" "foo" {}
`

const testAccDataSourceOpenIaaSStorageRepositoriesWithFilter = `
data "cloudtemple_compute_iaas_opensource_storage_repositories" "filtered" {
  machine_manager_id = "%s"
}
`
