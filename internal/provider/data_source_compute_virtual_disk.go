package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDisk() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceVirtualDiskRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceVirtualDiskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	disk, err := client.Compute().VirtualDisk().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, disk.ID)
	sw.set("virtual_machine_id", disk.VirtualMachineId)
	sw.set("machine_manager_id", disk.MachineManagerId)
	sw.set("name", disk.Name)
	sw.set("capacity", disk.Capacity)
	sw.set("disk_unit_number", disk.DiskUnitNumber)
	sw.set("controller_bus_number", disk.ControllerBusNumber)
	sw.set("datastore_id", disk.DatastoreId)
	sw.set("datastore_name", disk.DatastoreName)
	sw.set("instant_access", disk.InstantAccess)
	sw.set("native_id", disk.NativeId)
	sw.set("disk_path", disk.DiskPath)
	sw.set("provisioning_type", disk.ProvisioningType)
	sw.set("disk_mode", disk.DiskMode)
	sw.set("editable", disk.Editable)

	return sw.diags
}
