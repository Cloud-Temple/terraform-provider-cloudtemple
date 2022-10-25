package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceResourcePools() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceResourcePoolsRead,

		Schema: map[string]*schema.Schema{
			"resource_pools": {
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
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"moref": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"parent": {
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
									"cpu": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"max_usage": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"reservation_used": {
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
												"max_usage": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"reservation_used": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"ballooned_memory": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
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

func dataSourceResourcePoolsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	resourcePools, err := client.Compute().ResourcePool().List(ctx, "", "", "")
	if err != nil {
		return diag.FromErr(err)
	}

	res := make([]interface{}, len(resourcePools))

	for i, resourcePool := range resourcePools {
		res[i] = map[string]interface{}{
			"id":                 resourcePool.ID,
			"name":               resourcePool.Name,
			"machine_manager_id": resourcePool.MachineManagerID,
			"moref":              resourcePool.Moref,
			"parent": []interface{}{
				map[string]interface{}{
					"id":   resourcePool.Parent.ID,
					"type": resourcePool.Parent.Type,
				},
			},
			"metrics": []interface{}{
				map[string]interface{}{
					"cpu": []interface{}{
						map[string]interface{}{
							"max_usage":        resourcePool.Metrics.CPU.MaxUsage,
							"reservation_used": resourcePool.Metrics.CPU.ReservationUsed,
						},
					},
					"memory": []interface{}{
						map[string]interface{}{
							"max_usage":        resourcePool.Metrics.Memory.MaxUsage,
							"reservation_used": resourcePool.Metrics.Memory.ReservationUsed,
							"ballooned_memory": resourcePool.Metrics.Memory.BalloonedMemory,
						},
					},
				},
			},
		}
	}

	sw := newStateWriter(d, "resource-pools")
	sw.set("resource_pools", res)

	return sw.diags
}
