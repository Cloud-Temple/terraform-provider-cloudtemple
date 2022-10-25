package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceSnapshotsRead,

		Schema: map[string]*schema.Schema{
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"snapshots": {
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
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"create_time": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
		},
	}
}

func dataSourceSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	snapshots, err := client.Compute().Snapshot().List(ctx, d.Get("virtual_machine_id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(snapshots))
	for i, snapshot := range snapshots {
		res[i] = map[string]interface{}{
			"id":                 snapshot.ID,
			"virtual_machine_id": snapshot.VirtualMachineId,
			"name":               snapshot.Name,
			"create_time":        snapshot.CreateTime,
		}
	}

	sw := newStateWriter(d, "snapshots")
	sw.set("snapshots", res)

	return sw.diags
}
