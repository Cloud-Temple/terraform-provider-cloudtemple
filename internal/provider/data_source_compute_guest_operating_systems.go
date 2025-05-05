package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceGuestOperatingSystems() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of guest operating systems.",

		ReadContext: computeGuestOperatingSystemsRead,

		Schema: map[string]*schema.Schema{
			// In
			"host_cluster_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				AtLeastOneOf: []string{"host_cluster_id", "host_id"},
				Description:  "The ID of the host cluster to filter guest operating systems by. At least one of `host_cluster_id` or `host_id` must be specified.",
			},
			"host_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				AtLeastOneOf: []string{"host_cluster_id", "host_id"},
				Description:  "The ID of the host to filter guest operating systems by. At least one of `host_cluster_id` or `host_id` must be specified.",
			},
			"os_family": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Filter guest operating systems by OS family (e.g., 'Windows', 'Linux').",
			},
			"version": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotWhiteSpace,
				Description:  "Filter guest operating systems by version.",
			},

			// Out
			"guest_operating_systems": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of guest operating systems matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"moref": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The managed object reference ID of the guest operating system.",
						},
						"family": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The family of the guest operating system (e.g., Windows, Linux).",
						},
						"full_name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The full name of the guest operating system.",
						},
					},
				},
			},
		},
	}
}

// computeGuestOperatingSystemsRead lit les systèmes d'exploitation invités et les mappe dans le state Terraform
func computeGuestOperatingSystemsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les systèmes d'exploitation invités
	guestOSs, err := c.Compute().GuestOperatingSystem().List(ctx, &client.GuestOperatingSystemFilter{
		HostClusterID: d.Get("host_cluster_id").(string),
		HostID:        d.Get("host_id").(string),
		OsFamily:      d.Get("os_family").(string),
		Version:       d.Get("version").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("guest_operating_systems")

	// Mapper manuellement les données en utilisant la fonction helper
	tfGuestOSs := make([]map[string]interface{}, len(guestOSs))
	for i, guestOS := range guestOSs {
		tfGuestOSs[i] = helpers.FlattenGuestOperatingSystem(guestOS)
	}

	// Définir les données dans le state
	if err := d.Set("guest_operating_systems", tfGuestOSs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
