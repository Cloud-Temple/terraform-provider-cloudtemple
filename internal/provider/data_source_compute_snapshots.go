package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all snapshots for a specific virtual machine.",

		ReadContext: computeSnapshotsRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
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

// computeSnapshotsRead lit les snapshots et les mappe dans le state Terraform
func computeSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les snapshots
	virtualMachineId := d.Get("virtual_machine_id").(string)
	snapshots, err := c.Compute().Snapshot().List(ctx, &client.SnapshotFilter{
		VirtualMachineID: virtualMachineId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("snapshots")

	// Mapper manuellement les données en utilisant la fonction helper
	tfSnapshots := make([]map[string]interface{}, len(snapshots))
	for i, snapshot := range snapshots {
		tfSnapshots[i] = helpers.FlattenSnapshot(snapshot)
	}

	// Définir les données dans le state
	if err := d.Set("snapshots", tfSnapshots); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
