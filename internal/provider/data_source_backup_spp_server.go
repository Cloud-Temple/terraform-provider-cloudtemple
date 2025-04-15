package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSPPServer() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific backup SPP server.",

		ReadContext: backupSPPServerRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				Description:   "The ID of the SPP server to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the SPP server to retrieve. Conflicts with `id`.",
			},
			"tenant_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
				Description:   "The tenant ID to filter SPP servers by. Only used when searching by name.",
			},

			// Out
			"address": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The network address of the SPP server.",
			},
		},
	}
}

// backupSPPServerRead lit un serveur SPP de backup et le mappe dans le state Terraform
func backupSPPServerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var server *client.BackupSPPServer
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		server, err = c.Backup().SPPServer().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if server == nil {
			return diag.FromErr(fmt.Errorf("failed to find SPP server with id %q", id))
		}
	} else {
		// Obtenir la liste des serveurs SPP
		tenantId, err := getTenantID(ctx, c, d)
		if err != nil {
			return diag.FromErr(err)
		}
		servers, err := c.Backup().SPPServer().List(ctx, &client.BackupSPPServerFilter{
			TenantId: tenantId,
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to list SPP servers: %s", err))
		}

		// Recherche par name
		name := d.Get("name").(string)
		if name != "" {
			for _, s := range servers {
				if s.Name == name {
					server = s
					break
				}
			}
			if server == nil {
				return diag.FromErr(fmt.Errorf("failed to find SPP server with name %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(server.ID)

	// Mapper les données en utilisant la fonction helper
	serverData := helpers.FlattenBackupSPPServer(server)

	// Définir les données dans le state
	for k, v := range serverData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
