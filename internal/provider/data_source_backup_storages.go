package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupStorages() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			storages, err := client.Backup().Storage().List(ctx)
			return map[string]interface{}{
				"id":       "storages",
				"storages": storages,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"storages": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"resource_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"site": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"storage_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"host_address": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"port_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"ssl_connection": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"initialize_status": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"version": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"is_ready": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"capacity": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"free": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"total": {
										Type:     schema.TypeInt,
										Computed: true,
									},
									"update_time": {
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
