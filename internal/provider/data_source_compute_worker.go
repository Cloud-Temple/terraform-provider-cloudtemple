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
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
			},

			// Out
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
