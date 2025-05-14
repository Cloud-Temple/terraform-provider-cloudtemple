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
				Description:  "The ID of the virtual machine to retrieve controllers for.",
			},
			"types": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "Filter controllers by type. If not specified, all controller types will be returned.",
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
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual controllers for the specified virtual machine.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual controller.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine this controller belongs to.",
						},
						"hot_add_remove": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether devices can be added to or removed from this controller while the virtual machine is running.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the controller (e.g., SCSI, IDE, USB).",
						},
						"sub_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The sub-type of the controller, providing more specific information about the controller type.",
						},
						"label": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The label of the controller as displayed in the virtual machine settings.",
						},
						"summary": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "A summary description of the controller.",
						},
						"virtual_disks": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of virtual disk IDs attached to this controller.",

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
		Types:            helpers.GetStringList(d, "types"),
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
