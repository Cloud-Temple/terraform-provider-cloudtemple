package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceHost() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			// Recherche par nom
			name := d.Get("name").(string)
			if name != "" {
				hosts, err := client.Compute().Host().List(ctx, "", "", "", "")
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

			// Recherche par ID
			id := d.Get("id").(string)
			if id != "" {
				host, err := client.Compute().Host().Read(ctx, id)
				if err != nil {
					return nil, err
				}
				if host == nil {
					return nil, fmt.Errorf("failed to find host with id %q", id)
				}
				return host, nil
			}

			return nil, fmt.Errorf("either id or name must be specified")
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

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"machine_manager_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"metrics": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"esx": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"build": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"full_name": {
										Type:     schema.TypeString,
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
									"overall_cpu_usage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cpu_mhz": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cpu_cores": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"cpu_threads": {
										Type:     schema.TypeInt,
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
									"memory_size": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"memory_usage": {
										Type:     schema.TypeInt,
										Computed: true,
									},
								},
							},
						},
						"maintenance_status": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"uptime": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"connected": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
			"virtual_machines": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
