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
				Description:   "The ID of the virtual disk to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the virtual disk to retrieve. Conflicts with `id`.",
			},
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id", "template_id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "Filter virtual disks by the ID of the virtual machine they are attached to.",
			},
			"storage_repository_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "Filter virtual disks by the ID of the storage repository they are located on.",
			},
			"template_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id", "virtual_machine_id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "Filter virtual disks by the ID of the template they are attached to.",
			},
			"attachable": {
				Type:          schema.TypeBool,
				Optional:      true,
				ConflictsWith: []string{"id"},
				Description:   "Filter virtual disks by whether they can be attached to a virtual machine.",
			},

			// Out
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the virtual disk in the Open IaaS system.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the virtual disk.",
			},
			"size": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The size of the virtual disk in bytes.",
			},
			"usage": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The amount of space used on the virtual disk in bytes.",
			},
			"is_snapshot": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether the virtual disk is a snapshot.",
			},
			"virtual_machines": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual machines this disk is attached to.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual machine.",
						},
						"read_only": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the disk is attached in read-only mode to this virtual machine.",
						},
					},
				},
			},
			"templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of templates this disk is attached to.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the template.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the template.",
						},
						"read_only": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the disk is attached in read-only mode to this template.",
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
	// Pour les data sources, on ne gère pas le champ "connected" comme input, donc on passe une chaîne vide
	diskData := helpers.FlattenOpenIaaSVirtualDisk(disk, "")

	// Définir les données dans le state
	for k, v := range diskData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
