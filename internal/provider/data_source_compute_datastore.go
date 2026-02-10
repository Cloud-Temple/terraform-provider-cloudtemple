package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceDatastore() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific datastore.",

		ReadContext: computeDatastoreRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datastore to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the datastore to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the machine manager to filter datastores by. Only used when searching by name.",
			},
			"datacenter_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datacenter to filter datastores by. Only used when searching by name.",
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host to filter datastores by. Only used when searching by name.",
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the host cluster to filter datastores by. Only used when searching by name.",
			},
			"datastore_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				Default:       "",
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the datastore cluster to filter datastores by. Only used when searching by name.",
			},

			// Out
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
	}
}

// computeDatastoreRead lit un datastore et le mappe dans le state Terraform
func computeDatastoreRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var datastore *client.Datastore
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		datastores, err := c.Compute().Datastore().List(ctx, &client.DatastoreFilter{
			Name:               name,
			MachineManagerId:   d.Get("machine_manager_id").(string),
			DatacenterId:       d.Get("datacenter_id").(string),
			HostId:             d.Get("host_id").(string),
			HostClusterId:      d.Get("host_cluster_id").(string),
			DatastoreClusterId: d.Get("datastore_cluster_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find datastore named %q: %s", name, err))
		}
		for _, ds := range datastores {
			if ds.Name == name {
				datastore = ds
				break
			}
		}
		if datastore == nil {
			return diag.FromErr(fmt.Errorf("failed to find datastore named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			datastore, err = c.Compute().Datastore().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if datastore == nil {
				return diag.FromErr(fmt.Errorf("failed to find datastore with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(datastore.ID)

	// Mapper les données en utilisant la fonction helper
	datastoreData := helpers.FlattenDatastore(datastore)

	// Définir les données dans le state
	for k, v := range datastoreData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
