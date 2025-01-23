package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasNetwork() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific network from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var network *client.OpenIaaSNetwork
			name := d.Get("name").(string)
			if name != "" {
				networks, err := c.Compute().OpenIaaS().Network().List(ctx, &client.OpenIaaSNetworkFilter{
					MachineManagerID: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					diag.Errorf("failed to find network named %q: %s", name, err)
				}
				for _, currNetwork := range networks {
					if currNetwork.Name == name {
						network = currNetwork
					}
				}
				diag.Errorf("failed to find network named %q", name)
			} else {
				id := d.Get("id").(string)
				var err error
				network, err = c.Compute().OpenIaaS().Network().Read(ctx, id)
				if err == nil && network == nil {
					diag.Errorf("failed to find network with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(network.ID)
			d.Set("name", network.Name)
			d.Set("machine_manager_id", network.MachineManager.ID)
			d.Set("internal_id", network.InternalID)
			d.Set("pool", []interface{}{
				map[string]interface{}{
					"id":   network.Pool.ID,
					"name": network.Pool.Name,
				},
			})
			d.Set("maximum_transmission_unit", network.MaximumTransmissionUnit)
			d.Set("network_adapters", network.NetworkAdapters)
			d.Set("network_block_device", network.NetworkBlockDevice)
			d.Set("insecure_network_block_device", network.InsecureNetworkBlockDevice)

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
