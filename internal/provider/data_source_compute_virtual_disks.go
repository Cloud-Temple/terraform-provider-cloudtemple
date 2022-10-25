package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDisks() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualDisksRead,

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
			},
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

func dataSourceVirtualDisksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	disks, err := client.Compute().VirtualDisk().List(ctx, d.Get("virtual_machine_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(disks))
	for i, disk := range disks {
		res[i] = map[string]interface{}{
			"virtual_machine_id":    disk.VirtualMachineId,
			"machine_manager_id":    disk.MachineManagerId,
			"name":                  disk.Name,
			"capacity":              disk.Capacity,
			"disk_unit_number":      disk.DiskUnitNumber,
			"controller_bus_number": disk.ControllerBusNumber,
			"datastore_id":          disk.DatastoreId,
			"datastore_name":        disk.DatastoreName,
			"instant_access":        disk.InstantAccess,
			"native_id":             disk.NativeId,
			"disk_path":             disk.DiskPath,
			"provisioning_type":     disk.ProvisioningType,
			"disk_mode":             disk.DiskMode,
			"editable":              disk.Editable,
		}
	}

	sw := newStateWriter(d, "virtual-disks")
	sw.set("virtual_disks", res)

	return sw.diags
}
