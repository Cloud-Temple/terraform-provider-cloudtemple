package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSPPServers() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of backup SPP servers.",

		ReadContext: backupSPPServersRead,

		Schema: map[string]*schema.Schema{
			// In
			"tenant_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The tenant ID to filter SPP servers by.",
			},

			// Out
			"spp_servers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of SPP servers matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the SPP server.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the SPP server.",
						},
						"address": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The network address of the SPP server.",
						},
					},
				},
			},
		},
	}
}

// backupSPPServersRead lit les serveurs SPP de backup et les mappe dans le state Terraform
func backupSPPServersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer le tenant ID
	tenantId, err := getTenantID(ctx, c, d)
	if err != nil {
		return diag.FromErr(err)
	}

	// Récupérer les serveurs SPP
	sppServers, err := c.Backup().SPPServer().List(ctx, &client.BackupSPPServerFilter{
		TenantId: tenantId,
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("spp_servers")

	// Mapper manuellement les données en utilisant la fonction helper
	tfSPPServers := make([]map[string]interface{}, len(sppServers))
	for i, server := range sppServers {
		tfSPPServers[i] = helpers.FlattenBackupSPPServer(server)
	}

	// Définir les données dans le state
	if err := d.Set("spp_servers", tfSPPServers); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
