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

func dataSourceMarketplaceItem() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific item from the marketplace.",

		ReadContext: marketplaceItemRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the marketplace item to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				AtLeastOneOf:  []string{"id", "name"},
				ConflictsWith: []string{"id"},
				Description:   "The name of the marketplace item to retrieve. Conflicts with `id`.",
			},

			// Out
			"editor": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The editor/publisher of the marketplace item.",
			},
			"icon": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The icon URL of the marketplace item.",
			},
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the marketplace item (French).",
			},
			"description_en": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The description of the marketplace item (English).",
			},
			"creation_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the marketplace item.",
			},
			"last_update": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The last update date of the marketplace item.",
			},
			"categories": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The categories of the marketplace item.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"type": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The type of the marketplace item.",
			},
			"version": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The version of the marketplace item.",
			},
			"build": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The build of the marketplace item.",
			},
			"details": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Details about the marketplace item (French).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"overview": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Overview of the item.",
						},
						"how_to_use": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "How to use the item.",
						},
						"support": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Support information.",
						},
						"terms_and_conditions": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Terms and conditions.",
						},
					},
				},
			},
			"details_en": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Details about the marketplace item (English).",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"overview": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Overview of the item.",
						},
						"how_to_use": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "How to use the item.",
						},
						"support": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Support information.",
						},
						"terms_and_conditions": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "Terms and conditions.",
						},
					},
				},
			},
			"deployment_options": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "Deployment options for the marketplace item.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"targets": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of deployment targets.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The key of the target.",
									},
									"name": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The name of the target.",
									},
									"skus": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of SKUs available for this target.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
									"files": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "List of files available for this target.",
										Elem: &schema.Schema{
											Type: schema.TypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func marketplaceItemRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var item *client.MarketplaceItem
	var err error

	// Search by name
	name := d.Get("name").(string)
	if name != "" {
		items, err := c.Marketplace().Item().List(ctx)
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find marketplace item named %q: %s", name, err))
		}
		for _, i := range items {
			if i.Name == name && i.Type == "Virtual Machine Image" {
				item = i
				break
			}
		}
		if item == nil {
			return diag.FromErr(fmt.Errorf("failed to find marketplace item named %q", name))
		}
	} else {
		// Search by ID
		id := d.Get("id").(string)
		item, err = c.Marketplace().Item().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if item == nil {
			return diag.FromErr(fmt.Errorf("failed to find marketplace item with id %q", id))
		}
		if item.Type != "Virtual Machine Image" {
			return diag.FromErr(fmt.Errorf("marketplace item with id %q is not a Virtual Machine Image (type: %s)", id, item.Type))
		}
	}

	// Set the datasource ID
	d.SetId(item.ID)

	// Map the data using the helper function
	itemData := helpers.FlattenMarketplaceItem(item)

	// Set the data in the state
	for k, v := range itemData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
