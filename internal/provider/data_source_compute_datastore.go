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
		Description: "",

		ReadContext: computeDatastoreRead,

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
			"datastore_cluster_id": {
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
				Type:     schema.TypeBool,
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
