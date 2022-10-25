package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHostClusters() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceHostClustersRead,

		Schema: map[string]*schema.Schema{
			"host_clusters": {
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
				},
			},
		},
	}
}

func dataSourceHostClustersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	hcs, err := client.Compute().HostCluster().List(ctx, "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(hcs))
	for i, h := range hcs {
		hosts := make([]interface{}, len(h.Hosts))
		for j, host := range h.Hosts {
			hosts[j] = map[string]interface{}{
				"id":   host.ID,
				"type": host.Type,
			}
		}

		res[i] = map[string]interface{}{
			"id":    h.ID,
			"name":  h.Name,
			"moref": h.Moref,
			"hosts": hosts,
			"metrics": []interface{}{
				map[string]interface{}{
					"total_cpu":     h.Metrics.TotalCpu,
					"total_memory":  h.Metrics.TotalMemory,
					"total_storage": h.Metrics.TotalStorage,
					"cpu_used":      h.Metrics.CpuUsed,
					"memory_used":   h.Metrics.MemoryUsed,
					"storage_used":  h.Metrics.StorageUsed,
				},
			},
			"virtual_machines_number": h.VirtualMachinesNumber,
			"machine_manager_id":      h.MachineManagerId,
		}
	}

	sw := newStateWriter(d, "host-clusters")
	sw.set("host_clusters", res)

	return sw.diags
}
