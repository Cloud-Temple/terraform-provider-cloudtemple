package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceResourcePool() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: dataSourceResourcePoolRead,

		Schema: map[string]*schema.Schema{
			"id": {
				Type:     schema.TypeString,
				Required: true,
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
	}
}

func dataSourceResourcePoolRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	client := getClient(meta)

	resourcePool, err := client.Compute().ResourcePool().Read(ctx, d.Get("id").(string))
	if err != nil {
		return diag.FromErr(err)
	}

	sw := newStateWriter(d, resourcePool.ID)
	sw.set("id", resourcePool.ID)
	sw.set("name", resourcePool.Name)
	sw.set("machine_manager_id", resourcePool.MachineManagerID)
	sw.set("moref", resourcePool.Moref)
	sw.set("parent", []interface{}{
		map[string]interface{}{
			"id":   resourcePool.Parent.ID,
			"type": resourcePool.Parent.Type,
		},
	})
	sw.set("metrics", []interface{}{
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
	})

	return sw.diags
}
