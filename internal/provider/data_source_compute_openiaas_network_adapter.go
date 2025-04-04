package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasNetworkAdapter() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network adapter from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				adapters, err := c.Compute().OpenIaaS().NetworkAdapter().List(ctx, d.Get("virtual_machine_id").(string))
				if err != nil {
					return nil, fmt.Errorf("failed to find network adapter named %q: %s", name, err)
				}
				for _, adapter := range adapters {
					if adapter.Name == name {
						return adapter, nil
					}
				}
				return nil, fmt.Errorf("failed to find network adapter named %q", name)
			}

			id := d.Get("id").(string)
			if id != "" {
				adapter, err := c.Compute().OpenIaaS().NetworkAdapter().Read(ctx, id)
				if err == nil && adapter == nil {
					return nil, fmt.Errorf("failed to find network adapter with id %q", id)
				}
				return adapter, err
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
			"virtual_machine_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				RequiredWith:  []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},

			// Out
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
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
			"network": {
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
			"mac_address": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"mtu": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"attached": {
				Type:     schema.TypeBool,
				Computed: true,
			},
		},
	}
}
