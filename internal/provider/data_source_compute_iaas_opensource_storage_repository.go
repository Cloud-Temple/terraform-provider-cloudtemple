package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasStorageRepository() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific storage repository from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSStorageRepositoryRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the storage repository to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the storage repository to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager to filter storage repositories by. Required when searching by `name`.",
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the pool to filter storage repositories by.",
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host to filter storage repositories by.",
			},
			"type": {
				Type:        schema.TypeString,
				Optional:    true,
				Computed:    true,
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
				Description: "Filter storage repositories by whether they are shared or not.",
			},

			// Out
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the storage repository in the Open IaaS system.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the storage repository.",
			},
			"maintenance_mode": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the storage repository is in maintenance mode.",
			},
			"accessible": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "Indicates if the storage repository is accessible (1) or not (0).",
			},
			"free_capacity": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The available free space in the storage repository in bytes.",
			},
			"max_capacity": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The maximum capacity of the storage repository in bytes.",
			},
			"virtual_disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual disk IDs stored in this repository.",

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}

// computeOpenIaaSStorageRepositoryRead lit un dépôt de stockage OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSStorageRepositoryRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var repository *client.OpenIaaSStorageRepository
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		stringTypes := make([]string, 0, 1)
		if d.Get("type").(string) != "" {
			stringTypes = append(stringTypes, d.Get("type").(string))
		}

		repositories, err := c.Compute().OpenIaaS().StorageRepository().List(ctx, &client.StorageRepositoryFilter{
			MachineManagerId: d.Get("machine_manager_id").(string),
			PoolId:           d.Get("pool_id").(string),
			HostId:           d.Get("host_id").(string),
			StorageTypes:     stringTypes,
			Shared:           d.Get("shared").(bool),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find storage repository named %q: %s", name, err))
		}
		for _, sr := range repositories {
			if sr.Name == name {
				repository = sr
				break
			}
		}
		if repository == nil {
			return diag.FromErr(fmt.Errorf("failed to find storage repository named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			repository, err = c.Compute().OpenIaaS().StorageRepository().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if repository == nil {
				return diag.FromErr(fmt.Errorf("failed to find storage repository with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(repository.ID)

	// Mapper les données en utilisant la fonction helper
	repositoryData := helpers.FlattenOpenIaaSStorageRepository(repository)

	// Définir les données dans le state
	for k, v := range repositoryData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
