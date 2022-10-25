package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceVirtualDisks() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			disks, err := client.Compute().VirtualDisk().List(ctx, d.Get("virtual_machine_id").(string))
			return map[string]interface{}{
				"id":            "virtual_disks",
				"virtual_disks": disks,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Required: true,
			},

			// Out
			"virtual_disks": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"virtual_machine_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"capacity": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"disk_unit_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"controller_bus_number": {
							Type:     schema.TypeInt,
							Computed: true,
						},
						"datastore_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"datastore_name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"instant_access": {
							Type:     schema.TypeBool,
							Computed: true,
						},
						"native_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_path": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"provisioning_type": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"disk_mode": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"editable": {
							Type:     schema.TypeBool,
							Computed: true,
						},
					},
				},
			},
		},
	}
}
