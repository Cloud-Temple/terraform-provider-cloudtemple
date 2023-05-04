package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceWorkers() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			workers, err := client.Compute().Worker().List(ctx, "")
			return map[string]interface{}{
				"id":               "machine_managers",
				"machine_managers": workers,
			}, err
		}),

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
