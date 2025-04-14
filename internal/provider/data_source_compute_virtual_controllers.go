package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualControllers() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual controllers for a specific virtual machine.",

		ReadContext: computeVirtualControllersRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},
			"types": {
				Type:     schema.TypeList,
				Optional: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
					ValidateFunc: validation.StringInSlice([]string{
						"USB2",
						"USB3",
						"SCSI",
						"IDE",
						"PCI",
						"CD/DVD",
						"NVME",
					}, false),
				},
			},

			// Out
			"virtual_controllers": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machine_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"hot_add_remove": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"sub_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"label": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"summary": {
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
					},
				},
			},
		},
	}
}

// computeVirtualControllersRead lit les contrôleurs virtuels et les mappe dans le state Terraform
func computeVirtualControllersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les contrôleurs virtuels
	virtualMachineId := d.Get("virtual_machine_id").(string)
	controllers, err := c.Compute().VirtualController().List(ctx, &client.VirtualControllerFilter{
		VirtualMachineId: virtualMachineId,
		Types:            GetStringList(d, "types"),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("virtual_controllers")

	// Mapper manuellement les données en utilisant la fonction helper
	tfControllers := make([]map[string]interface{}, len(controllers))
	for i, controller := range controllers {
		tfControllers[i] = helpers.FlattenVirtualController(controller)
		tfControllers[i]["id"] = controller.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_controllers", tfControllers); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
