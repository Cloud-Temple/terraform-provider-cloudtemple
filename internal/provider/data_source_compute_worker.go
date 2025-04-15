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

func dataSourceWorker() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific worker (vCenter) from the infrastructure.",

		ReadContext: dataSourceWorkerRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the worker to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the worker to retrieve. Conflicts with `id`.",
			},

			// Out
			"full_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The full name of the worker (vCenter), including version information.",
			},
			"vendor": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The vendor of the worker (e.g., VMware).",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the worker software.",
			},
			"build": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The build number of the worker software.",
			},
			"locale_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The locale version of the worker software.",
			},
			"locale_build": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The locale build number of the worker software.",
			},
			"os_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The operating system type of the worker.",
			},
			"product_line_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The product line identifier of the worker.",
			},
			"api_type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of API provided by the worker.",
			},
			"api_version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the API provided by the worker.",
			},
			"instance_uuid": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The UUID of the worker instance.",
			},
			"license_product_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the licensed product.",
			},
			"license_product_version": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The version of the licensed product.",
			},
			"tenant_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the tenant that owns the worker.",
			},
			"tenant_name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the tenant that owns the worker.",
			},
		},
	}
}

// dataSourceWorkerRead lit un worker et le mappe dans le state Terraform
func dataSourceWorkerRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var worker *client.Worker
	var err error

	// Recherche par ID
	id := d.Get("id").(string)
	if id != "" {
		worker, err = c.Compute().Worker().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if worker == nil {
			return diag.FromErr(fmt.Errorf("failed to find worker with id %q", id))
		}
	} else {
		// Recherche par nom
		name := d.Get("name").(string)
		if name != "" {
			workers, err := c.Compute().Worker().List(ctx, name)
			if err != nil {
				return diag.FromErr(fmt.Errorf("failed to list workers: %s", err))
			}
			for _, w := range workers {
				if w.Name == name {
					worker = w
					break
				}
			}
			if worker == nil {
				return diag.FromErr(fmt.Errorf("failed to find worker with name %q", name))
			}
		} else {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
	}

	// Définir l'ID de la datasource
	d.SetId(worker.ID)

	// Mapper les données en utilisant la fonction helper
	workerData := helpers.FlattenWorker(worker)

	// Définir les données dans le state
	for k, v := range workerData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
