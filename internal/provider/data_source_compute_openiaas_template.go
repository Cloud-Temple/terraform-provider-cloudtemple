package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific template from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var template *client.OpenIaasTemplate

			name := d.Get("name").(string)
			if name != "" {
				templates, err := c.Compute().OpenIaaS().Template().List(ctx, &client.OpenIaaSTemplateFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
					PoolId:           d.Get("pool_id").(string),
				})
				if err != nil {
					diag.Errorf("failed to find template named %q: %s", name, err)
				}
				for _, currTemplate := range templates {
					if currTemplate.Name == name {
						template = currTemplate
					}
				}
				diag.Errorf("failed to find template named %q", name)
			} else {
				id := d.Get("id").(string)
				template, err := c.Compute().OpenIaaS().Template().Read(ctx, id)
				if err == nil && template == nil {
					diag.Errorf("failed to find template with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(template.ID)

			d.Set("name", template.Name)
			d.Set("machine_manager_id", template.MachineManager.ID)
			d.Set("internal_id", template.InternalID)
			d.Set("cpu", template.CPU)
			d.Set("num_cores_per_socket", template.NumCoresPerSocket)
			d.Set("memory", template.Memory)
			d.Set("power_state", template.PowerState)
			d.Set("snapshots", template.Snapshots)
			d.Set("sla_policies", template.SLAPolicies)
			d.Set("disks", flattenDisks(template.Disks))
			d.Set("network_adapters", flattenNetworkAdapters(template.NetworkAdapters))

			diag.Errorf("network_adapters: %v", template.NetworkAdapters)

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
			"pool_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
			},

			// Out
			"internal_id": {
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
			"power_state": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"snapshots": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"sla_policies": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"disks": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"description": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"size": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"storage_repository": {
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
				},
			},
			"network_adapters": {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mac_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"mtu": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"attached": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"network": {
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
				},
			},
		},
	}
}

func flattenDisks(disks []client.TemplateDisk) []interface{} {
	if disks != nil {
		result := make([]interface{}, len(disks))
		for i, disk := range disks {
			result[i] = map[string]interface{}{
				"name":               disk.Name,
				"description":        disk.Description,
				"size":               disk.Size,
				"storage_repository": flattenBaseObject(disk.StorageRepository),
			}
		}
		return result
	}
	return make([]interface{}, 0)
}

func flattenNetworkAdapters(adapters []client.TemplateNetworkAdapter) []interface{} {
	if adapters != nil {
		result := make([]interface{}, len(adapters))
		for i, adapter := range adapters {
			result[i] = map[string]interface{}{
				"name":        adapter.Name,
				"mac_address": adapter.MacAddress,
				"mtu":         adapter.MTU,
				"attached":    adapter.Attached,
				"network":     flattenBaseObject(adapter.Network),
			}
		}
		return result
	}
	return make([]interface{}, 0)
}
