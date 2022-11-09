package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				dcs, err := client.Compute().VirtualDatacenter().List(ctx, "", "")
				if err != nil {
					return nil, fmt.Errorf("failed to find virtual datacenter named %q: %s", name, err)
				}
				for _, dc := range dcs {
					if dc.Name == name {
						return dc, nil
					}
				}
				return nil, fmt.Errorf("failed to find virtual datacenter named %q", name)
			}

			id := d.Get("id").(string)
			datacenter, err := client.Compute().VirtualDatacenter().Read(ctx, id)
			if err == nil && datacenter == nil {
				return nil, fmt.Errorf("failed to find virtual datacenter with id %q", id)
			}
			return datacenter, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
			},

			// Out
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
