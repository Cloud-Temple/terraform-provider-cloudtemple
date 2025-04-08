package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSPPServer() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				server, err := client.Backup().SPPServer().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if server == nil {
					return nil, fmt.Errorf("failed to find SPP server with id %q", id)
				}
				return server, nil
			}

			// Obtenir la liste des serveurs SPP
			tenantId, err := getTenantID(ctx, client, d)
			if err != nil {
				return nil, err
			}
			servers, err := client.Backup().SPPServer().List(ctx, tenantId)
			if err != nil {
				return nil, fmt.Errorf("failed to list SPP servers: %s", err)
			}

			// Recherche par name
			name := d.Get("name").(string)
			if name != "" {
				for _, server := range servers {
					if server.Name == name {
						return server, nil
					}
				}
				return nil, fmt.Errorf("failed to find SPP server with name %q", name)
			}

			return nil, fmt.Errorf("either id or name must be specified")
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"tenant_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"id"},
			},

			// Out
			"address": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
