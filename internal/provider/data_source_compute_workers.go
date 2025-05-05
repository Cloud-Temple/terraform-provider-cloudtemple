package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWorkers() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all workers (vCenters) from the infrastructure.",

		ReadContext: dataSourceWorkersRead,

		Schema: map[string]*schema.Schema{
			// Out
			"machine_managers": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of all workers (vCenters) in the infrastructure.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the worker (vCenter).",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the worker (vCenter).",
						},
						"full_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The full name of the worker (vCenter), including version information.",
						},
						"vendor": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The vendor of the worker (e.g., VMware).",
						},
						"version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the worker software.",
						},
						"build": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The build number of the worker software.",
						},
						"locale_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The locale version of the worker software.",
						},
						"locale_build": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The locale build number of the worker software.",
						},
						"os_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The operating system type of the worker.",
						},
						"product_line_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The product line identifier of the worker.",
						},
						"api_type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of API provided by the worker.",
						},
						"api_version": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The version of the API provided by the worker.",
						},
						"instance_uuid": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The UUID of the worker instance.",
						},
						"license_product_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the licensed product.",
						},
						"license_product_version": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The version of the licensed product.",
						},
						"tenant_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the tenant that owns the worker.",
						},
						"tenant_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the tenant that owns the worker.",
						},
					},
				},
			},
		},
	}
}

// dataSourceWorkersRead lit les workers et les mappe dans le state Terraform
func dataSourceWorkersRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les workers
	workers, err := c.Compute().Worker().List(ctx, "")
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("machine_managers")

	// Mapper manuellement les données en utilisant la fonction helper
	tfWorkers := make([]map[string]interface{}, len(workers))
	for i, worker := range workers {
		tfWorkers[i] = helpers.FlattenWorker(worker)
	}

	// Définir les données dans le state
	if err := d.Set("machine_managers", tfWorkers); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
