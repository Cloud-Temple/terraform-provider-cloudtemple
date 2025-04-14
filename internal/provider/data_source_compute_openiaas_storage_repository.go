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
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
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
				Type:     schema.TypeBool,
				Optional: true,
			},

			// Out
			"internal_id": {
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
			"accessible": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"free_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"max_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"virtual_disks": {
				Type:     schema.TypeList,
				Computed: true,

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
