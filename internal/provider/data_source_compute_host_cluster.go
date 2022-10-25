package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostCluster() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceHostClusterRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
			},
			"name": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"hosts": {
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
			"metrics": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"total_cpu": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_memory": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"total_storage": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"cpu_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"memory_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"storage_used": {
							Type:     schema.TypeInt,
							Computed: true,
						},
					},
				},
			},
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

func dataSourceHostClusterRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	hc, err := client.Compute().HostCluster().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	hosts := make([]interface{}, len(hc.Hosts))
	for j, host := range hc.Hosts {
		hosts[j] = map[string]interface{}{
			"id":   host.ID,
			"type": host.Type,
		}
	}

	sw := newStateWriter(d, hc.ID)
	sw.set("id", hc.ID)
	sw.set("name", hc.Name)
	sw.set("moref", hc.Moref)
	sw.set("hosts", hosts)
	sw.set("metrics", []interface{}{
		map[string]interface{}{
			"total_cpu":     hc.Metrics.TotalCpu,
			"total_memory":  hc.Metrics.TotalMemory,
			"total_storage": hc.Metrics.TotalStorage,
			"cpu_used":      hc.Metrics.CpuUsed,
			"memory_used":   hc.Metrics.MemoryUsed,
			"storage_used":  hc.Metrics.StorageUsed,
		},
	})
	sw.set("virtual_machines_number", hc.VirtualMachinesNumber)
	sw.set("machine_manager_id", hc.MachineManagerId)

	return sw.diags
}
