package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceHosts() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceHostsRead,

		Schema: map[string]*schema.Schema{
			"hosts": {
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
						"machine_manager_id": {
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
				},
			},
		},
	}
}

func dataSourceHostsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	hosts, err := client.Compute().Host().List(ctx, "", "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(hosts))
	for i, host := range hosts {
		virtualMachines := make([]interface{}, len(host.VirtualMachines))
		for j, vm := range host.VirtualMachines {
			virtualMachines[j] = map[string]interface{}{
				"id":   vm.ID,
				"type": vm.Type,
			}
		}

		res[i] = map[string]interface{}{
			"id":                 host.ID,
			"name":               host.Name,
			"moref":              host.Moref,
			"machine_manager_id": host.MachineManagerID,
			"metrics": []interface{}{
				map[string]interface{}{
					"esx": []interface{}{
						map[string]interface{}{
							"version":   host.Metrics.ESX.Version,
							"build":     host.Metrics.ESX.Build,
							"full_name": host.Metrics.ESX.FullName,
						},
					},
					"cpu": []interface{}{
						map[string]interface{}{
							"overall_cpu_usage": host.Metrics.CPU.OverallCPUUsage,
							"cpu_mhz":           host.Metrics.CPU.CPUMhz,
							"cpu_cores":         host.Metrics.CPU.CPUCores,
							"cpu_threads":       host.Metrics.CPU.CPUThreads,
						},
					},
					"memory": []interface{}{
						map[string]interface{}{
							"memory_size":  host.Metrics.Memory.MemorySize,
							"memory_usage": host.Metrics.Memory.MemoryUsage,
						},
					},
					"maintenance_status": host.Metrics.MaintenanceStatus,
					"uptime":             host.Metrics.Uptime,
					"connected":          host.Metrics.Connected,
				},
			},
			"virtual_machines": virtualMachines,
		}
	}

	sw := newStateWriter(d, "hosts")
	sw.set("hosts", res)

	return sw.diags
}
