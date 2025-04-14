package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceDatastores() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeDatastoresRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"machine_manager_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datacenter_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"host_cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"host_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"datastore_cluster_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

			// Out
			"datastores": {
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
							Type:     schema.TypeBool,
							Computed: true,
						},
						"unique_id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"machine_manager_id": {
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
				},
			},
		},
	}
}

// computeDatastoresRead lit les datastores et les mappe dans le state Terraform
func computeDatastoresRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les datastores
	datastores, err := c.Compute().Datastore().List(ctx, &client.DatastoreFilter{
		Name:               d.Get("name").(string),
		MachineManagerId:   d.Get("machine_manager_id").(string),
		DatacenterId:       d.Get("datacenter_id").(string),
		HostClusterId:      d.Get("host_cluster_id").(string),
		HostId:             d.Get("host_id").(string),
		DatastoreClusterId: d.Get("datastore_cluster_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("datastores")

	// Mapper manuellement les données en utilisant la fonction helper
	tfDatastores := make([]map[string]interface{}, len(datastores))
	for i, datastore := range datastores {
		tfDatastores[i] = helpers.FlattenDatastore(datastore)
	}

	// Définir les données dans le state
	if err := d.Set("datastores", tfDatastores); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
