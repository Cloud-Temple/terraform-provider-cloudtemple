package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualControllers() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualControllersRead,

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
			},
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

func dataSourceVirtualControllersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	controllers, err := client.Compute().VirtualController().List(ctx, d.Get("virtual_machine_id").(string), "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(controllers))
	for i, controller := range controllers {
		res[i] = map[string]interface{}{
			"id":                 controller.ID,
			"virtual_machine_id": controller.VirtualMachineId,
			"hot_add_remove":     controller.HotAddRemove,
			"type":               controller.Type,
			"label":              controller.Label,
			"summary":            controller.Summary,
			"virtual_disks":      controller.VirtualDisks,
		}
	}

	sw := newStateWriter(d, "virtual-controllers")
	sw.set("virtual_controllers", res)

	return sw.diags
}
