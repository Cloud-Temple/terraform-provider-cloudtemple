package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasMachineManager() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve an Availability Zone.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			var machineManager *client.OpenIaaSMachineManager
			var err error
			id := d.Get("id").(string)
			if id != "" {
				machineManager, err = c.Compute().OpenIaaS().MachineManager().Read(ctx, id)
				if err == nil && machineManager == nil {
					return nil, fmt.Errorf("failed to find machine manager with id %q", id)
				}
				return machineManager, err
			}

			name := d.Get("name").(string)
			if name != "" {
				machineManagers, err := c.Compute().OpenIaaS().MachineManager().List(ctx)
				if err != nil {
					return nil, fmt.Errorf("failed to list machine managers: %s", err)
				}
				for _, machineManager := range machineManagers {
					if machineManager.Name == name {
						return machineManager, nil
					}
				}
				return nil, fmt.Errorf("failed to find machine manager named %q", name)
			}

			return nil, fmt.Errorf("either id or name must be specified")
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"os_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"os_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"xoa_version": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
