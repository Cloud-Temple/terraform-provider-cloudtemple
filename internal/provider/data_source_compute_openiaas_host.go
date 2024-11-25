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
					diag.Errorf("failed to find host named %q: %s", name, err)
				}
				for _, currHost := range hosts {
					if currHost.Name == name {
						host = currHost
					}
				}
				diag.Errorf("failed to find host named %q", name)
			} else {
				id := d.Get("id").(string)
				host, err := c.Compute().OpenIaaS().Host().Read(ctx, id)
				if err == nil && host == nil {
					diag.Errorf("failed to find host with id %q", id)
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
			d.Set("memory", []interface{}{
				map[string]interface{}{
					"usage": host.Memory.Usage,
					"size":  host.Memory.Size,
				},
			})
			d.Set("cpu", []interface{}{
				map[string]interface{}{
					"cores":   host.Cpu.Cores,
					"sockets": host.Cpu.Sockets,
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
						"cores": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"sockets": {
							Type:     schema.TypeInt,
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
		},
	}
}
