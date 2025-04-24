package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasStorageRepositories() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all storage repositories from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSStorageRepositoriesRead,

		Schema: map[string]*schema.Schema{
			// In
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter storage repositories by machine manager ID.",
			},
			"pool_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter storage repositories by pool ID.",
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter storage repositories by host ID.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Available values are: ext, lvm, lvmoiscsi, lvmohba, nfs, smb, iso, nfs_iso, cifs",
				ValidateFunc: validation.StringInSlice([]string{
					"ext",
					"lvm",
					"lvmoiscsi",
					"lvmohba",
					"nfs",
					"smb",
					"iso",
					"nfs_iso",
					"cifs",
				}, true),
			},
			"shared": {
				Type:        schema.TypeBool,
				Optional:    true,
				Default:     false,
				Description: "Filter storage repositories by whether they are shared or not.",
			},

			// Out
			"storage_repositories": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of storage repositories matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the storage repository.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the storage repository.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the storage repository in the Open IaaS system.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this storage repository belongs to.",
						},
						"pool_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the pool this storage repository belongs to.",
						},
						"host_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the host this storage repository is attached to.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the storage repository.",
						},
						"maintenance_status": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the storage repository is in maintenance mode.",
						},
						"max_capacity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum capacity of the storage repository in bytes.",
						},
						"free_capacity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The available free space in the storage repository in bytes.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the storage repository (ext, lvm, nfs, etc.).",
						},
						"virtual_disks": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of virtual disk IDs stored in this repository.",
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"shared": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the storage repository is shared across multiple hosts.",
						},
						"accessible": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Indicates if the storage repository is accessible (1) or not (0).",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSStorageRepositoriesRead lit les dépôts de stockage OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSStorageRepositoriesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Préparer les filtres
	stringTypes := make([]string, 0, 1)
	if d.Get("type").(string) != "" {
		stringTypes = append(stringTypes, d.Get("type").(string))
	}

	// Récupérer les dépôts de stockage OpenIaaS
	repositories, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, &client.StorageRepositoryFilter{
		MachineManagerId: d.Get("machine_manager_id").(string),
		PoolId:           d.Get("pool_id").(string),
		HostId:           d.Get("host_id").(string),
		StorageTypes:     stringTypes,
		Shared:           d.Get("shared").(bool),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_storage_repositories")

	// Mapper manuellement les données en utilisant la fonction helper
	tfRepositories := make([]map[string]interface{}, len(repositories))
	for i, repository := range repositories {
		tfRepositories[i] = helpers.FlattenOpenIaaSStorageRepository(repository)
	}

	// Définir les données dans le state
	if err := d.Set("storage_repositories", tfRepositories); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
