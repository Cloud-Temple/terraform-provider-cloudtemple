package provider

import (
	"context"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceMarketplaceItems() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of items from the marketplace.",

		ReadContext: readMarketplaceItems,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Description: "Filter items by name (case-insensitive partial match).",
			},

			// Out
			"marketplace_items": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of marketplace items matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the marketplace item.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the marketplace item.",
						},
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
				},
			},
		},
	}
}

func readMarketplaceItems(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Retrieve marketplace items
	items, err := c.Marketplace().Item().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Filter to keep only Virtual Machine Image items
	vmImageItems := []*client.MarketplaceItem{}
	for _, item := range items {
		if item.Type == "Virtual Machine Image" {
			vmImageItems = append(vmImageItems, item)
		}
	}
	items = vmImageItems

	// Filter by name if provided
	name := d.Get("name").(string)
	if name != "" {
		filteredItems := []*client.MarketplaceItem{}
		lowerName := strings.ToLower(name)
		for _, item := range items {
			if strings.Contains(strings.ToLower(item.Name), lowerName) {
				filteredItems = append(filteredItems, item)
			}
		}
		items = filteredItems
	}

	// Set the datasource ID
	d.SetId("marketplace_items")

	// Map the data using the helper function
	marketplaceItems := make([]map[string]interface{}, len(items))
	for i, item := range items {
		marketplaceItems[i] = helpers.FlattenMarketplaceItem(item)
	}

	// Set the data in the state
	if err := d.Set("marketplace_items", marketplaceItems); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
