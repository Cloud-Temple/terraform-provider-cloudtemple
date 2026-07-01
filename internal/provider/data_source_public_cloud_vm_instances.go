package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMInstances() *schema.Resource {
	// Each listed element is a Computed id plus the shared computed attributes.
	elem := publicCloudVMInstanceComputedAttributes()
	elem["id"] = &schema.Schema{Type: schema.TypeString, Computed: true, Description: "The unique identifier of the VM instance."}

	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM instances of the tenant, optionally filtered. The full result set is paginated automatically.",

		ReadContext: publicCloudVMInstancesRead,

		Schema: map[string]*schema.Schema{
			// In — filters
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by exact VM name.",
			},
			"status": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter by status (e.g. `running`, `stopped`).",
			},
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter by availability zone ID.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter by instance family ID.",
			},
			"order_by": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"name", "createdAt", "updatedAt", "status", "vcpu", "ramGb"}, false),
				Description:  "Field to order by (`name`, `createdAt`, `updatedAt`, `status`, `vcpu`, `ramGb`).",
			},
			"order_dir": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"asc", "desc"}, false),
				Description:  "Order direction (`asc` or `desc`).",
			},

			// Out
			"instances": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of VM instances.",
				Elem:        &schema.Resource{Schema: elem},
			},
		},
	}
}

func publicCloudVMInstancesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	vms, err := c.PublicCloudVM().Instance().List(ctx, &client.PublicCloudVMInstanceFilter{
		Name:               d.Get("name").(string),
		Status:             d.Get("status").(string),
		AvailabilityZoneID: d.Get("availability_zone_id").(string),
		FamilyID:           d.Get("instance_family_id").(string),
		OrderBy:            d.Get("order_by").(string),
		OrderDir:           d.Get("order_dir").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_instances")

	tfInstances := make([]map[string]interface{}, len(vms))
	for i, vm := range vms {
		tfInstances[i] = helpers.FlattenPublicCloudVMInstance(vm)
	}

	if err := d.Set("instances", tfInstances); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
