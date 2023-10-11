package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceVirtualSwitch() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				virtualSwitches, err := c.Compute().VirtualSwitch().List(ctx, &client.VirtualSwitchFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
					DatacenterId:     d.Get("datacenter_id").(string),
					HostClusterId:    d.Get("host_cluster_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find virtual switch named %q: %s", name, err)
				}
				for _, dvs := range virtualSwitches {
					if dvs.Name == name {
						return dvs, nil
					}
				}
				return nil, fmt.Errorf("failed to find virtual switch named %q", name)
			}

			id := d.Get("id").(string)
			virtualSwitch, err := c.Compute().VirtualSwitch().Read(ctx, id)
			if err == nil && virtualSwitch == nil {
				return nil, fmt.Errorf("failed to find virtual switch with id %q", id)
			}
			return virtualSwitch, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"folder_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
