package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				networks, err := c.Compute().OpenIaaS().Network().List(ctx, &client.OpenIaaSNetworkFilter{
					MachineManagerID: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find network named %q: %s", name, err)
				}
				for _, network := range networks {
					if network.Name == name {
						return network, nil
					}
				}
				return nil, fmt.Errorf("failed to find network named %q", name)
			}

			id := d.Get("id").(string)
			if id != "" {
				network, err := c.Compute().OpenIaaS().Network().Read(ctx, id)
				if err != nil || network == nil {
					return nil, fmt.Errorf("failed to find network with id %q", id)
				}
				return network, err
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
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"pool": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"maximum_transmission_unit": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"network_adapters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"network_block_device": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"insecure_network_block_device": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}
