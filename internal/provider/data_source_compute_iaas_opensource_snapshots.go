package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasSnapshots() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all snapshots from an Open IaaS infrastructure for a specific virtual machine.",

		ReadContext: computeOpenIaaSSnapshotsRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the virtual machine to retrieve snapshots for.",
			},

			// Out
			"snapshots": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of snapshots for the specified virtual machine.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the snapshot.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the snapshot.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the snapshot.",
						},
						"virtual_machine_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the virtual machine this snapshot belongs to.",
						},
						"create_time": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The timestamp when the snapshot was created.",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSSnapshotsRead lit les snapshots OpenIaaS et les mappe dans le state Terraform
func computeOpenIaaSSnapshotsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les snapshots OpenIaaS
	virtualMachineId := d.Get("virtual_machine_id").(string)
	snapshots, err := c.Compute().OpenIaaS().Snapshot().List(ctx, &client.OpenIaaSSnapshotFilter{
		VirtualMachineID: virtualMachineId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("openiaas_snapshots_" + virtualMachineId)

	// Mapper manuellement les données en utilisant la fonction helper
	tfSnapshots := make([]map[string]interface{}, len(snapshots))
	for i, snapshot := range snapshots {
		tfSnapshots[i] = helpers.FlattenOpenIaaSSnapshot(snapshot)
	}

	// Définir les données dans le state
	if err := d.Set("snapshots", tfSnapshots); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
