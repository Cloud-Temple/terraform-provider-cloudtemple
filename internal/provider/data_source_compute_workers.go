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
						"full_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"vendor": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"build": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"locale_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"locale_build": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"os_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"product_line_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"api_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"api_version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instance_uuid": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"license_product_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"license_product_version": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"tenant_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"tenant_name": {
							Type:     schema.TypeString,
							Computed: true,
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
