package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasHost() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific host from an Open IaaS infrastructure.",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			name := d.Get("name").(string)
			if name != "" {
				hosts, err := c.Compute().OpenIaaS().Host().List(ctx, &client.OpenIaasHostFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return nil, fmt.Errorf("failed to find host named %q: %s", name, err)
				}
				for _, host := range hosts {
					if host.Name == name {
						return host, nil
					}
				}
				return nil, fmt.Errorf("failed to find host named %q", name)
			}

			id := d.Get("id").(string)
			if id != "" {
				var err error
				host, err := c.Compute().OpenIaaS().Host().Read(ctx, id)
				if err == nil && host == nil {
					return nil, fmt.Errorf("failed to find host with id %q", id)
				}
				return host, err
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
			"machine_manager_type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_name": {
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
				},
			},
			"master": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"uptime": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"update_data": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"maintenance_mode": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"status": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
			"reboot_required": {
				Type:     schema.TypeBool,
				Computed: true,
			},
			"virtual_machines": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"metrics": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"xoa": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"full_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"build": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
						"memory": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"usage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"size": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"cpu": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"sockets": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cores": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"model": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"model_name": {
										Type:     schema.TypeString,
										Computed: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
