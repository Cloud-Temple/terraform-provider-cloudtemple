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

func dataSourceOpenIaasVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual disk from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSVirtualDiskRead,

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
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},
			"storage_repository_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},
			"template_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
			},
			"attachable": {
				Type:          schema.TypeBool,
				Optional:      true,
				ConflictsWith: []string{"id"},
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
			"size": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"usage": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"is_snapshot": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"virtual_machines": {
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
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"templates": {
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
						"read_only": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSVirtualDiskRead lit un disque virtuel OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var disk *client.OpenIaaSVirtualDisk
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		disks, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, &client.OpenIaaSVirtualDiskFilter{
			StorageRepositoryID: d.Get("storage_repository_id").(string),
			TemplateID:          d.Get("template_id").(string),
			VirtualMachineID:    d.Get("virtual_machine_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual disk named %q: %s", name, err))
		}
		for _, d := range disks {
			if d.Name == name {
				disk = d
				break
			}
		}
		if disk == nil {
			return diag.FromErr(fmt.Errorf("failed to find virtual disk named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			disk, err = c.Compute().OpenIaaS().VirtualDisk().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if disk == nil {
				return diag.FromErr(fmt.Errorf("failed to find virtual disk with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(disk.ID)

	// Mapper les données en utilisant la fonction helper
	diskData := helpers.FlattenOpenIaaSVirtualDisk(disk)

	// Définir les données dans le state
	for k, v := range diskData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
