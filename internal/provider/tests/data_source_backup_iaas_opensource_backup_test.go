package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

const (
	OpenIaaSBackupId = "BACKUP_IAAS_OPENSOURCE_BACKUP_ID"
)

func TestAccDataSourceOpenIaaSBackup(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: providerFactories,
		Steps: []resource.TestStep{
			{
				Config: fmt.Sprintf(testAccDataSourceOpenIaaSBackup, os.Getenv(OpenIaaSBackupId)),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backup.foo", "id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backup.foo", "internal_id"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backup.foo", "mode"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backup.foo", "size"),
					resource.TestCheckResourceAttrSet("data.cloudtemple_backup_iaas_opensource_backup.foo", "timestamp"),
				),
			},
			{
				Config:      testAccDataSourceOpenIaaSBackupMissing,
				ExpectError: regexp.MustCompile("failed to find backup with id"),
			},
		},
	})
}

const testAccDataSourceOpenIaaSBackup = `
data "cloudtemple_backup_iaas_opensource_backup" "foo" {
  id = "%s"
}
`

const testAccDataSourceOpenIaaSBackupMissing = `
data "cloudtemple_backup_iaas_opensource_backup" "foo" {
  id = "12345678-1234-5678-1234-567812345678"
}
`
