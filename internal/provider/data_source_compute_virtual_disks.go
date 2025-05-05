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
				Description:  "The ID of the virtual machine to retrieve disks for.",
			},

			// Out
			"virtual_disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of virtual disks attached to the specified virtual machine.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the virtual disk.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine this disk is attached to.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager (vCenter) where this virtual disk is located.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the virtual disk.",
						},
						"capacity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The capacity of the virtual disk in Bytes.",
						},
						"disk_unit_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The unit number of the disk on its controller.",
						},
						"controller_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the controller this disk is attached to.",
						},
						"controller_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the controller (e.g., SCSI, IDE, NVME).",
						},
						"controller_bus_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The bus number of the controller.",
						},
						"datastore_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the datastore where this virtual disk is stored.",
						},
						"datastore_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the datastore where this virtual disk is stored.",
						},
						"instant_access": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the disk is an instant access disk.",
						},
						"native_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The native ID of the disk in the hypervisor.",
						},
						"disk_path": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The path to the disk file in the datastore.",
						},
						"provisioning_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The provisioning type of the disk",
						},
						"disk_mode": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The disk mode",
						},
						"editable": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Whether the disk is editable.",
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
	d.SetId("virtual_disks" + virtualMachineId)

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
