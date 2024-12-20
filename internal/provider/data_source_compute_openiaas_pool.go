package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasPool() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific pool from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var pool *client.OpenIaasPool
			name := d.Get("name").(string)
			if name != "" {
				pools, err := c.Compute().OpenIaaS().Pool().List(ctx, &client.OpenIaasPoolFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					diag.Errorf("failed to find pool named %q: %s", name, err)
				}
				for _, currPool := range pools {
					if currPool.Name == name {
						pool = currPool
					}
				}
				diag.Errorf("failed to find pool named %q", name)
			} else {
				id := d.Get("id").(string)
				pool, err := c.Compute().OpenIaaS().Pool().Read(ctx, id)
				if err == nil && pool == nil {
					diag.Errorf("failed to find pool with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(pool.ID)

			d.Set("name", pool.Name)
			d.Set("machine_manager_id", pool.MachineManager.ID)
			d.Set("internal_id", pool.InternalID)
			d.Set("high_availability_enabled", pool.HighAvailabilityEnabled)
			d.Set("cpu", []interface{}{
				map[string]interface{}{
					"cores":   pool.Cpu.Cores,
					"sockets": pool.Cpu.Sockets,
				},
			})
			d.Set("hosts", pool.Hosts)
			d.Set("type", []interface{}{
				map[string]interface{}{
					"key":         pool.Type.Key,
					"description": pool.Type.Description,
				},
			})

			return sw.diags
		},

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
