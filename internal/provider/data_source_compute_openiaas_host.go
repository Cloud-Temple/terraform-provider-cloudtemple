package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasHost() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific host from an Open IaaS infrastructure.",

		ReadContext: func(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
			c := getClient(meta)
			var host *client.OpenIaaSHost
			name := d.Get("name").(string)
			if name != "" {
				hosts, err := c.Compute().OpenIaaS().Host().List(ctx, &client.OpenIaasHostFilter{
					MachineManagerId: d.Get("machine_manager_id").(string),
				})
				if err != nil {
					return diag.Errorf("failed to find host named %q: %s", name, err)
				}
				for _, currHost := range hosts {
					if currHost.Name == name {
						host = currHost
					}
				}
				if host == nil {
					return diag.Errorf("failed to find host named %q", name)
				}
			} else {
				id := d.Get("id").(string)
				var err error
				host, err = c.Compute().OpenIaaS().Host().Read(ctx, id)
				if err == nil && host == nil {
					return diag.Errorf("failed to find host with id %q", id)
				}
			}

			sw := newStateWriter(d)

			d.SetId(host.ID)
			d.Set("name", host.Name)
			d.Set("machine_manager_id", host.MachineManager.ID)
			d.Set("internal_id", host.InternalId)
			d.Set("pool", []interface{}{
				map[string]interface{}{
					"id":   host.Pool.ID,
					"name": host.Pool.Name,
					"type": []interface{}{
						map[string]interface{}{
							"key":         host.Pool.Type.Key,
							"description": host.Pool.Type.Description,
						},
					},
				},
			})
			d.Set("uptime", host.Uptime)
			d.Set("power_state", host.PowerState)
			d.Set("update_data", []interface{}{
				map[string]interface{}{
					"maintenance_mode": host.UpdateData.MaintenanceMode,
					"status":           host.UpdateData.Status,
				},
			})
			d.Set("metrics", []interface{}{
				map[string]interface{}{
					"xoa": []interface{}{
						map[string]interface{}{
							"version":   host.Metrics.XOA.Version,
							"full_name": host.Metrics.XOA.FullName,
							"build":     host.Metrics.XOA.Build,
						},
					},
					"memory": []interface{}{
						map[string]interface{}{
							"usage": host.Metrics.Memory.Usage,
							"size":  host.Metrics.Memory.Size,
						},
					},
					"cpu": []interface{}{
						map[string]interface{}{
							"cores":      host.Metrics.Cpu.Cores,
							"sockets":    host.Metrics.Cpu.Sockets,
							"model":      host.Metrics.Cpu.Model,
							"model_name": host.Metrics.Cpu.ModelName,
						},
					},
				},
			})
			d.Set("reboot_required", host.RebootRequired)
			d.Set("virtual_machines", host.VirtualMachines)

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
