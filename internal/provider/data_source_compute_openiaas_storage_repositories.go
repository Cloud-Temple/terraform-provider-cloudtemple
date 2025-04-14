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
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"pool_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
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
				Type:     schema.TypeBool,
				Optional: true,
				Default:  false,
			},

			// Out
			"storage_repositories": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"pool_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"maintenance_status": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"max_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"free_capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_disks": {
							Type:     schema.TypeList,
							Computed: true,
							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"shared": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"accessible": {
							Type:     schema.TypeInt,
							Computed: true,
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
