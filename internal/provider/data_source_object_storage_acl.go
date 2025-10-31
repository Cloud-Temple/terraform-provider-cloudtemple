package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceACL() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve ACL information for object storage. You must specify either `bucket_name` to list storage accounts with access to a bucket, or `storage_account_name` to list buckets accessible by a storage account.",

		ReadContext: dataSourceACLRead,

		Schema: map[string]*schema.Schema{
			// In - One of these two must be provided
			"bucket_name": {
				Description:   "The name of the bucket to list storage accounts for. Conflicts with `storage_account_name`.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"storage_account_name"},
			},
			"storage_account_name": {
				Description:   "The name of the storage account to list buckets for. Conflicts with `bucket_name`.",
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"bucket_name"},
			},

			// Out
			"acls": {
				Description: "The list of ACL entries. When filtering by bucket, this contains storage accounts. When filtering by storage account, this contains buckets.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the ACL entry.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the resource (storage account or bucket).",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"role": {
							Description: "The role/permission level (e.g., 'owner').",
							Type:        schema.TypeString,
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

// dataSourceACLRead lit les ACLs et les mappe dans le state Terraform
func dataSourceACLRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var acls []*client.ACL
	var err error

	bucketName := d.Get("bucket_name").(string)
	storageAccountName := d.Get("storage_account_name").(string)

	// Valider qu'exactement un des deux paramètres est fourni
	if bucketName == "" && storageAccountName == "" {
		return diag.Errorf("either bucket_name or storage_account_name must be specified")
	}

	if bucketName != "" && storageAccountName != "" {
		return diag.Errorf("only one of bucket_name or storage_account_name can be specified")
	}

	// Récupérer les ACLs selon le filtre
	if bucketName != "" {
		acls, err = c.ObjectStorage().ACL().ListByBucket(ctx, bucketName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error listing ACLs for bucket %s: %s", bucketName, err))
		}
		d.SetId(fmt.Sprintf("bucket:%s", bucketName))
	} else {
		acls, err = c.ObjectStorage().ACL().ListByStorageAccount(ctx, storageAccountName)
		if err != nil {
			return diag.FromErr(fmt.Errorf("error listing ACLs for storage account %s: %s", storageAccountName, err))
		}
		d.SetId(fmt.Sprintf("storage_account:%s", storageAccountName))
	}

	// Mapper les données en utilisant la fonction helper
	tfACLs := make([]map[string]interface{}, len(acls))
	for i, acl := range acls {
		tfACLs[i] = helpers.FlattenACL(acl)
	}

	// Définir les données dans le state
	if err := d.Set("acls", tfACLs); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
