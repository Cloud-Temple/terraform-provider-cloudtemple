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
			},
			"template_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"virtual_machine_id"},
			},
			"storage_repository_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"attachable": {
				Type:     schema.TypeBool,
				Optional: true,
			},

			// Out
			"virtual_disks": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"internal_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
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
						"storage_repository_id": {
							Type:     schema.TypeString,
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
		tfDisks[i] = helpers.FlattenOpenIaaSVirtualDisk(disk)
	}

	// Définir les données dans le state
	if err := d.Set("virtual_disks", tfDisks); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
