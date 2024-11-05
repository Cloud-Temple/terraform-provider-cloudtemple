package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasVirtualMachine() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific virtual machine from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				virtualMachines, err := c.Compute().OpenIaaS().VirtualMachine().List(ctx, &client.OpenIaaSVirtualMachineFilter{
					MachineManagerID: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find virtual machine named %q: %s", name, err)
				}
				for _, virtualMachine := range virtualMachines {
					if virtualMachine.Name == name {
						return virtualMachine, nil
					}
				}
				return nil, fmt.Errorf("failed to find virtual machine named %q", name)
			}

			id := d.Get("id").(string)
			virtualMachine, err := c.Compute().OpenIaaS().VirtualMachine().Read(ctx, id)
			if err == nil && virtualMachine == nil {
				return nil, fmt.Errorf("failed to find virtual machine with id %q", id)
			}
			return virtualMachine, err
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
			"machine_manager": {
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
			"internal_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"secure_boot": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"auto_power_on": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"dvd_drive": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"boot_order": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"operating_system_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"cpu": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"num_cores_per_socket": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"memory": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"addresses": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"ipv6": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"ipv4": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
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
			"host": {
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
		},
	}
}
