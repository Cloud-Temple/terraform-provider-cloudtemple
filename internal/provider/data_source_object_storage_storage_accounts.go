package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStorageAccounts() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all storage accounts in the object storage.",

		ReadContext: dataSourceStorageAccountsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"storage_accounts": {
				Description: "The list of storage accounts.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"access_key_id": {
							Description: "The access key ID of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"arn": {
							Description: "The ARN of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"create_date": {
							Description: "The creation date of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"path": {
							Description: "The path of the storage account.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"tags": {
							Description: "The tags associated with the storage account.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"key": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"value": {
										Type:     schema.TypeString,
										Computed: true,
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

func dataSourceStorageAccountsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer tous les storage accounts
	accounts, err := c.ObjectStorage().StorageAccount().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("storage_accounts")

	// Mapper les données en utilisant la fonction helper
	tfAccounts := make([]map[string]interface{}, len(accounts))
	for i, account := range accounts {
		tfAccounts[i] = helpers.FlattenStorageAccount(account)
	}

	// Définir les données dans le state
	if err := d.Set("storage_accounts", tfAccounts); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
