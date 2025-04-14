package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualDisks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all virtual disks for a specific virtual machine.",

		ReadContext: dataSourceVirtualDisksRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
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
						"virtual_machine_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"disk_unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"controller_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"controller_bus_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"datastore_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instant_access": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"native_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provisioning_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"editable": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

// dataSourceVirtualDisksRead lit les disques virtuels et les mappe dans le state Terraform
func dataSourceVirtualDisksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les disques virtuels
	virtualMachineId := d.Get("virtual_machine_id").(string)
	disks, err := c.Compute().VirtualDisk().List(ctx, &client.VirtualDiskFilter{
		VirtualMachineID: virtualMachineId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("virtual_disks")

	// Mapper manuellement les données en utilisant la fonction helper
	tfDisks := make([]map[string]interface{}, len(disks))
	for i, disk := range disks {
		tfDisks[i] = helpers.FlattenVirtualDisk(disk)
		tfDisks[i]["id"] = disk.ID
	}

	// Définir les données dans le state
	if err := d.Set("virtual_disks", tfDisks); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
