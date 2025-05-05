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

func dataSourceCompany() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific company.",

		ReadContext: dataSourceCompanyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the company to retrieve.",
			},

			// Out
			"name": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The name of the company.",
			},
		},
	}
}

// dataSourceCompanyRead lit une company et la mappe dans le state Terraform
func dataSourceCompanyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer l'ID de la company
	id := d.Get("id").(string)

	// Récupérer la company
	company, err := c.IAM().Company().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if company == nil {
		return diag.FromErr(fmt.Errorf("failed to find company with id %q", id))
	}

	// Définir l'ID de la datasource
	d.SetId(company.ID)

	// Mapper les données en utilisant la fonction helper
	companyData := helpers.FlattenCompany(company)

	// Définir les données dans le state
	for k, v := range companyData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
