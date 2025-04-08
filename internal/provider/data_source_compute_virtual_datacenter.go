package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualDatacenter() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				datacenters, err := c.Compute().VirtualDatacenter().List(ctx, &client.VirtualDatacenterFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find virtual datacenter named %q: %s", name, err)
				}
				for _, datacenter := range datacenters {
					if datacenter.Name == name {
						return datacenter, nil
					}
				}
				return nil, fmt.Errorf("failed to find virtual datacenter named %q", name)
			}

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				datacenter, err := c.Compute().VirtualDatacenter().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if datacenter == nil {
					return nil, fmt.Errorf("failed to find virtual datacenter with id %q", id)
				}
				return datacenter, nil
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
			"vcenter": {
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
			"tenant_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
