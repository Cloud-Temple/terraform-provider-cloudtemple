package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceACLEntry() *schema.Resource {
	return &schema.Resource{
		Description: "Manage ACL entries for object storage buckets. An ACL entry grants a specific role to a storage account on a bucket.",

		CreateContext: objectStorageACLEntryCreate,
		ReadContext:   objectStorageACLEntryRead,
		DeleteContext: objectStorageACLEntryDelete,

		Schema: map[string]*schema.Schema{
			// In
			"bucket": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the bucket.",
			},
			"role": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The role to grant.",
			},
			"storage_account": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the storage account.",
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the ACL entry (format: bucket/role/storage_account).",
			},
		},
	}
}

func objectStorageACLEntryCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	bucket := d.Get("bucket").(string)
	role := d.Get("role").(string)
	storageAccount := d.Get("storage_account").(string)

	activityId, err := c.ObjectStorage().ACLEntry().Grant(ctx, bucket, role, storageAccount)
	if err != nil {
		return diag.Errorf("failed to grant ACL entry: %s", err)
	}

	if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to grant ACL entry: %s", err)
	}

	// Set ID as a composite of bucket/role/storageAccount
	d.SetId(fmt.Sprintf("%s/%s/%s", bucket, role, storageAccount))

	return objectStorageACLEntryRead(ctx, d, meta)
}

func objectStorageACLEntryRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// ACL entries don't have a dedicated Read endpoint
	// We keep the resource in state as long as it hasn't been deleted
	// If the entry doesn't exist on the server, the Delete operation will handle it
	return nil
}

func objectStorageACLEntryDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	bucket := d.Get("bucket").(string)
	role := d.Get("role").(string)
	storageAccount := d.Get("storage_account").(string)

	activityId, err := c.ObjectStorage().ACLEntry().Revoke(ctx, bucket, role, storageAccount)
	if err != nil {
		return diag.Errorf("failed to revoke ACL entry: %s", err)
	}

	if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to revoke ACL entry: %s", err)
	}

	return nil
}
