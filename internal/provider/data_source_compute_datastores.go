package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastores() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of datastores.",

		ReadContext: computeDatastoresRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter datastores by name.",
			},
			"machine_manager_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter datastores by machine manager ID.",
			},
			"datacenter_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter datastores by datacenter ID.",
			},
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter datastores by host cluster ID.",
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter datastores by host ID.",
			},
			"datastore_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter datastores by datastore cluster ID.",
			},

			// Out
			"datastores": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of datastores matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the datastore.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the datastore.",
						},
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the datastore.",
						},
						"max_capacity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The maximum capacity of the datastore in bytes.",
						},
						"free_capacity": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The free capacity of the datastore in bytes.",
						},
						"accessible": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "Indicates whether the datastore is accessible (1 for accessible, 0 for not accessible).",
						},
						"maintenance_status": {
							Type:        schema.TypeBool,
							Computed:    true,
							Deprecated:  "Use maintenance_mode instead. This field will be removed in a future version.",
							Description: "Indicates whether the datastore is in maintenance mode. Deprecated: use maintenance_mode instead.",
						},
						"maintenance_mode": {
							Type:        schema.TypeBool,
							Computed:    true,
							Description: "Indicates whether the datastore is in maintenance mode.",
						},
						"unique_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the datastore in the infrastructure.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager this datastore belongs to.",
						},
						"type": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The type of the datastore (e.g., VMFS, NFS).",
						},
						"virtual_machines_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of virtual machines using this datastore.",
						},
						"hosts_number": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of hosts that have access to this datastore.",
						},
						"hosts_names": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of host names that have access to this datastore.",

							Elem: &schema.Schema{
								Type: schema.TypeString,
							},
						},
						"associated_folder": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The folder associated with this datastore.",
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
