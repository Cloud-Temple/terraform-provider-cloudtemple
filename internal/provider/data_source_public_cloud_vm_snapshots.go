package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve the snapshots of a Public Cloud VM instance.",

		ReadContext: publicCloudVMSnapshotsRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the VM whose snapshots are listed.",
			},

			// Out
			"snapshots": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of snapshots of the VM.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                 {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the snapshot."},
						"virtual_machine_id": {Type: schema.TypeString, Computed: true, Description: "The ID of the VM the snapshot belongs to."},
						"name":               {Type: schema.TypeString, Computed: true, Description: "The name of the snapshot."},
						"status":             {Type: schema.TypeString, Computed: true, Description: "The status of the snapshot."},
						"created_at":         {Type: schema.TypeString, Computed: true, Description: "The creation date of the snapshot (RFC3339)."},
					},
				},
			},
		},
	}
}

func publicCloudVMSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	vmID := d.Get("virtual_machine_id").(string)

	snaps, err := c.PublicCloudVM().Snapshot().List(ctx, vmID)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId(vmID)

	tfSnaps := make([]map[string]interface{}, len(snaps))
	for i, s := range snaps {
		tfSnaps[i] = helpers.FlattenPublicCloudVMSnapshot(s)
	}
	if err := d.Set("snapshots", tfSnaps); err != nil {
		return diag.FromErr(err)
	}
	return nil
}
