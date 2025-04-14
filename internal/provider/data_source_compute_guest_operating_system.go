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

func dataSourceGuestOperatingSystem() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: computeGuestOperatingSystemRead,

		Schema: map[string]*schema.Schema{
			// In
			"moref": {
				Type:     schema.TypeString,
				Required: true,
			},
			"host_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"host_cluster_id"},
				AtLeastOneOf:  []string{"host_id", "host_cluster_id"},
			},
			"host_cluster_id": {
				Type:          schema.TypeString,
				Optional:      true,
				ValidateFunc:  validation.IsUUID,
				ConflictsWith: []string{"host_id"},
				AtLeastOneOf:  []string{"host_id", "host_cluster_id"},
			},

			// Out
			"family": {
				Type:     schema.TypeString,
				Computed: true,
			},
			"full_name": {
				Type:     schema.TypeString,
				Computed: true,
			},
		},
	}
}

// computeGuestOperatingSystemRead lit un système d'exploitation invité et le mappe dans le state Terraform
func computeGuestOperatingSystemRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer le système d'exploitation invité
	guestOS, err := c.Compute().GuestOperatingSystem().Read(ctx, d.Get("moref").(string), &client.GuestOperatingSystemFilter{
		HostID:        d.Get("host_id").(string),
		HostClusterID: d.Get("host_cluster_id").(string),
	})

	if err != nil {
		return diag.FromErr(err)
	}
	if guestOS == nil {
		return diag.FromErr(fmt.Errorf("failed to find guest operating system"))
	}

	// Définir l'ID de la datasource
	d.SetId(guestOS.Moref)

	// Mapper les données en utilisant la fonction helper
	guestOSData := helpers.FlattenGuestOperatingSystem(guestOS)

	// Définir les données dans le state
	for k, v := range guestOSData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
