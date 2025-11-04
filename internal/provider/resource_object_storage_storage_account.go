package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceStorageAccount() *schema.Resource {
	return &schema.Resource{
		Description: "Create and manage object storage storage accounts.",

		CreateContext: objectStorageStorageAccountCreate,
		ReadContext:   objectStorageStorageAccountRead,
		DeleteContext: objectStorageStorageAccountDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the storage account.",
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the storage account.",
			},
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID of the storage account.",
			},
			"access_secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret access key of the storage account. This is only available during creation.",
			},
			"arn": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ARN of the storage account.",
			},
			"create_date": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The creation date of the storage account.",
			},
			"path": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The path of the storage account.",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The tags associated with the storage account.",
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

func objectStorageStorageAccountCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	createResp, err := c.ObjectStorage().StorageAccount().Create(ctx, &client.CreateStorageAccountRequest{
		Name: d.Get("name").(string),
	})
	if err != nil {
		return diag.Errorf("failed to create storage account: %s", err)
	}

	// Set ID - the name is the ID
	d.SetId(d.Get("name").(string))

	// Set the sensitive keys from creation response
	d.Set("access_key_id", createResp.AccessKeyID)
	d.Set("access_secret_key", createResp.SecretAccessKey)

	return objectStorageStorageAccountRead(ctx, d, meta)
}

func objectStorageStorageAccountRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	storageAccount, err := c.ObjectStorage().StorageAccount().Read(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to read storage account: %s", err)
	}
	if storageAccount == nil {
		d.SetId("")
		return nil
	}

	// Use helper to flatten the storage account data
	storageAccountData := helpers.FlattenStorageAccount(storageAccount)

	// Add tags
	tags := make([]map[string]interface{}, len(storageAccount.Tags))
	for i, tag := range storageAccount.Tags {
		tags[i] = map[string]interface{}{
			"key":   tag.Key,
			"value": tag.Value,
		}
	}
	storageAccountData["tags"] = tags

	// Set all data in state (except sensitive keys which are only available at creation)
	for k, v := range storageAccountData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func objectStorageStorageAccountDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.ObjectStorage().StorageAccount().Delete(ctx, d.Id())
	if err != nil {
		return diag.Errorf("failed to delete storage account: %s", err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete storage account: %s", err)
	}

	return nil
}
