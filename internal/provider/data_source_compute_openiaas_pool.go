package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasPool() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific pool from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				pools, err := c.Compute().OpenIaaS().Pool().List(ctx, &client.OpenIaasPoolFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find pool named %q: %s", name, err)
				}
				for _, pool := range pools {
					if pool.Name == name {
						return pool, nil
					}
				}
			}

			id := d.Get("id").(string)
			if id != "" {
				id := d.Get("id").(string)
				var err error
				pool, err := c.Compute().OpenIaaS().Pool().Read(ctx, id)
				if err == nil && pool == nil {
					return nil, fmt.Errorf("failed to find pool with id %q", id)
				}
				return pool, err
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
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"high_availability_enabled": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"cpu": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cores": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"sockets": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosts": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
