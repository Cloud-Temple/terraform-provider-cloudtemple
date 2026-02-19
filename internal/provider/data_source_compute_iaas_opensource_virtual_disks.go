package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasVirtualDisks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual disks from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSVirtualDisksRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"template_id"},
				Description:   "Filter virtual disks by the ID of the virtual machine they are attached to. Conflicts with `template_id`.",
			},
			"template_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"virtual_machine_id"},
				Description:   "Filter virtual disks by the ID of the template they are attached to. Conflicts with `virtual_machine_id`.",
			},
			"storage_repository_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter virtual disks by the ID of the storage repository they are located on.",
			},
			"attachable": {
				Type:        schema.TypeBool,
				Optional:    true,
				Description: "Filter virtual disks by whether they can be attached to a virtual machine.",
			},

			// Out
			"virtual_disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual disks matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual disk.",
						},
						"internal_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The internal identifier of the virtual disk in the Open IaaS system.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual disk.",
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
						"storage_repository_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the storage repository where the disk is located.",
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
				},
			},
		},
	}
}

// computeOpenIaaSVirtualDisksRead lit les disques virtuels OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSVirtualDisksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les disques virtuels OpenIaaS
	disks, err := c.Compute().OpenIaaS().VirtualDisk().List(ctx, &client.OpenIaaSVirtualDiskFilter{
		StorageRepositoryID: d.Get("storage_repository_id").(string),
		TemplateID:          d.Get("template_id").(string),
		VirtualMachineID:    d.Get("virtual_machine_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_virtual_disks")

	// Mapper manuellement les données en utilisant la fonction helper
	tfDisks := make([]map[string]interface{}, len(disks))
	for i, disk := range disks {
		// Pour les data sources de liste, on ne gère pas le champ "connected" comme input, donc on passe une chaîne vide
		tfDisks[i] = helpers.FlattenOpenIaaSVirtualDisk(disk, "")
		tfDisks[i]["id"] = disk.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_disks", tfDisks); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
