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
		Description: "Used to retrieve a list of backup storage systems.",

		ReadContext: backupStoragesRead,

		Schema: map[string]*schema.Schema{
			// Out
			"storages": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of backup storage systems.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the storage system.",
						},
						"resource_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The resource type of the storage system.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the storage system.",
						},
						"site": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The site associated with the storage system.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the storage system.",
						},
						"storage_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The storage ID within the backup system.",
						},
						"host_address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The host address of the storage system.",
						},
						"port_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The port number used to connect to the storage system.",
						},
						"ssl_connection": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether SSL is used for the connection.",
						},
						"initialize_status": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The initialization status of the storage system.",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the storage system software.",
						},
						"is_ready": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether the storage system is ready for use.",
						},
						"capacity": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Capacity information for the storage system.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The amount of free space available in the storage system (in bytes).",
									},
									"total": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The total capacity of the storage system (in bytes).",
									},
									"update_time": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The timestamp when the capacity information was last updated.",
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
