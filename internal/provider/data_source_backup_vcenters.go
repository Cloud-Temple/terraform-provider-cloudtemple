package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupVCenters() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of vCenter servers registered with a backup SPP server.",

		ReadContext: backupVCentersRead,

		Schema: map[string]*schema.Schema{
			// In
			"spp_server_id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the SPP server to retrieve vCenters from.",
			},

			// Out
			"vcenters": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of vCenter servers registered with the specified SPP server.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the vCenter server.",
						},
						"internal_id": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The internal ID of the vCenter server in the backup system.",
						},
						"instance_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The instance ID of the vCenter server.",
						},
						"spp_server_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the SPP server this vCenter is registered with.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the vCenter server.",
						},
					},
				},
			},
		},
	}
}

// backupVCentersRead lit les vcenters de backup et les mappe dans le state Terraform
func backupVCentersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer le SPP server ID
	sppServerId := d.Get("spp_server_id").(string)

	// Récupérer les vcenters
	vcenters, err := c.Backup().VCenter().List(ctx, &client.BackupVCenterFilter{
		SppServerId: sppServerId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("vcenters")

	// Mapper manuellement les données en utilisant la fonction helper
	tfVCenters := make([]map[string]interface{}, len(vcenters))
	for i, vcenter := range vcenters {
		tfVCenters[i] = helpers.FlattenBackupVCenter(vcenter)
	}

	// Définir les données dans le state
	if err := d.Set("vcenters", tfVCenters); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
