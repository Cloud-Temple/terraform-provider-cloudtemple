package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceStorageAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific storage account.",

		ReadContext: dataSourceStorageAccountRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Description: "The name of the storage account.",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
			"id": {
				Description: "The ID of the storage account.",
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
	}
}

func dataSourceStorageAccountRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var account *client.StorageAccount
	var err error

	accountName := d.Get("name").(string)

	// Récupérer le storage account par son nom
	account, err = c.ObjectStorage().StorageAccount().Read(ctx, accountName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading storage account with name %s: %s", accountName, err))
	}

	// Définir l'ID de la datasource
	d.SetId(account.ID)

	// Mapper les données en utilisant la fonction helper
	accountData := helpers.FlattenStorageAccount(account)

	// Définir les données dans le state
	for k, v := range accountData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
