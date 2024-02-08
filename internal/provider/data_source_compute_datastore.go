package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastore() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, c *client.Client, d *schema.ResourceData, sw *stateWriter) (interface{}, error) {
			return getBy(
				ctx,
				d,
				"datastore",
				func(id string) (any, error) {
					return c.Compute().Datastore().Read(ctx, id)
				},
				func(d *schema.ResourceData) (any, error) {
					return c.Compute().Datastore().List(ctx, &client.DatastoreFilter{
						Name:             d.Get("name").(string),
						MachineManagerId: d.Get("machine_manager_id").(string),
						DatacenterId:     d.Get("datacenter_id").(string),
						HostId:           d.Get("host_id").(string),
						HostClusterId:    d.Get("host_cluster_id").(string),
					})
				},
				[]string{"name"},
			)
		}),

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},

			// Out
			"moref": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"max_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"free_capacity": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"accessible": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"maintenance_status": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"unique_id": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"type": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"virtual_machines_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hosts_number": {
				Type:     schema.TypeInt,
				Computed: true,
			},
			"hosts_names": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"associated_folder": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}
