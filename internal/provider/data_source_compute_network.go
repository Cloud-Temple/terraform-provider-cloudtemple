package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				networks, err := c.Compute().Network().List(ctx, &client.NetworkFilter{
					Name:             name,
					MachineManagerId: d.Get("machine_manager_id").(string),
					DatacenterId:     d.Get("datacenter_id").(string),
					VirtualMachineId: d.Get("virtual_machine_id").(string),
					Type:             d.Get("type").(string),
					VirtualSwitchId:  d.Get("virtual_switch_id").(string),
					HostId:           d.Get("host_id").(string),
					FolderId:         d.Get("folder_id").(string),
					HostClusterId:    d.Get("host_cluster_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find virtual network named %q: %s", name, err)
				}
				for _, n := range networks {
					if n.Name == name {
						return n, nil
					}
				}
				return nil, fmt.Errorf("failed to find virtual network named %q", name)
			}

			id := d.Get("id").(string)
			network, err := c.Compute().Network().Read(ctx, id)
			if err == nil && network == nil {
				return nil, fmt.Errorf("failed to find virtual network with id %q", id)
			}
			return network, err
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
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"type": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringInSlice([]string{"Network", "DistributedVirtualPortgroup"}, false),
			},
			"virtual_switch_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
			},
			"folder_id": {
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
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"host_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"host_names": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
		},
	}
}
