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

func dataSourceOpenIaasPool() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific pool from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSPoolRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the pool to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the pool to retrieve. Conflicts with `id`.",
			},
			"machine_manager_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The ID of the machine manager to filter pools by. Required when searching by `name`.",
			},

			// Out
			"internal_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The internal identifier of the pool in the Open IaaS system.",
			},
			"high_availability_enabled": {
				Type:        schema.TypeBool,
				Computed:    true,
				Description: "Whether high availability is enabled for this pool.",
			},
			"cpu": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "CPU information for the pool.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"cores": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of CPU cores in the pool.",
						},
						"sockets": {
							Type:        schema.TypeInt,
							Computed:    true,
							Description: "The number of CPU sockets in the pool.",
						},
					},
				},
			},
			"hosts": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of host IDs in this pool.",

				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Information about the pool type.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The key identifier of the pool type.",
						},
						"description": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The description of the pool type.",
						},
					},
				},
			},
		},
	}
}

// computeOpenIaaSPoolRead lit un pool OpenIaaS et le mappe dans le state Terraform
func computeOpenIaaSPoolRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var pool *client.OpenIaasPool
	var err error

	// Recherche par nom
	name := d.Get("name").(string)
	if name != "" {
		pools, err := c.Compute().OpenIaaS().Pool().List(ctx, &client.OpenIaasPoolFilter{
			MachineManagerId: d.Get("machine_manager_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find pool named %q: %s", name, err))
		}
		for _, p := range pools {
			if p.Name == name {
				pool = p
				break
			}
		}
		if pool == nil {
			return diag.FromErr(fmt.Errorf("failed to find pool named %q", name))
		}
	} else {
		// Recherche par ID
		id := d.Get("id").(string)
		if id != "" {
			pool, err = c.Compute().OpenIaaS().Pool().Read(ctx, id)
			if err != nil {
				return diag.FromErr(err)
			}
			if pool == nil {
				return diag.FromErr(fmt.Errorf("failed to find pool with id %q", id))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(pool.ID)

	// Mapper les données en utilisant la fonction helper
	poolData := helpers.FlattenOpenIaaSPool(pool)

	// Définir les données dans le state
	for k, v := range poolData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
