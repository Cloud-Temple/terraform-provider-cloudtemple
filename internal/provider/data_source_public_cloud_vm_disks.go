package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMDisks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve the disks (system and data) of a Public Cloud VM instance.",

		ReadContext: publicCloudVMDisksRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VM whose disks are listed.",
			},

			// Out
			"disks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of disks attached to the VM.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":           {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the disk."},
						"position":     {Type: schema.TypeInt, Computed: true, Description: "The position of the disk (0 is the system disk; data disks are 1+)."},
						"name":         {Type: schema.TypeString, Computed: true, Description: "The name (label) of the disk."},
						"size_gb":      {Type: schema.TypeInt, Computed: true, Description: "The size of the disk in GB."},
						"storage_type": {Type: schema.TypeString, Computed: true, Description: "The ID of the storage type."},
						"is_primary":   {Type: schema.TypeBool, Computed: true, Description: "Whether this is the primary (system) disk."},
					},
				},
			},
		},
	}
}

func publicCloudVMDisksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	vmID := d.Get("virtual_machine_id").(string)

	disks, err := c.PublicCloudVM().Disk().List(ctx, vmID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(vmID)

	tfDisks := make([]map[string]interface{}, len(disks))
	for i, disk := range disks {
		tfDisks[i] = helpers.FlattenPublicCloudVMDisk(disk)
	}
	if err := d.Set("disks", tfDisks); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
