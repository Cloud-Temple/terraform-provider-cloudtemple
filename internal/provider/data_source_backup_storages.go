package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupStorages() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: backupStoragesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"storages": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"site": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"storage_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ssl_connection": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"initialize_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"is_ready": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"capacity": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"update_time": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

// backupStoragesRead lit les storages de backup et les mappe dans le state Terraform
func backupStoragesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les storages
	storages, err := c.Backup().Storage().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("storages")

	// Mapper manuellement les données en utilisant la fonction helper
	tfStorages := make([]map[string]interface{}, len(storages))
	for i, storage := range storages {
		tfStorages[i] = helpers.FlattenBackupStorage(storage)
	}

	// Définir les données dans le state
	if err := d.Set("storages", tfStorages); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
