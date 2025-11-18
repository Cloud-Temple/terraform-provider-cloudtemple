package provider

import (
	"context"
	"fmt"
	"regexp"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Create and manage object storage buckets.",

		CreateContext: objectStorageBucketCreate,
		ReadContext:   objectStorageBucketRead,
		UpdateContext: objectStorageBucketUpdate,
		DeleteContext: objectStorageBucketDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Required:    true,
				ForceNew:    true,
				Description: "The name of the bucket. Must be alphanumeric (with hyphens and underscores) and up to 255 characters.",
				ValidateFunc: validation.All(
					validation.StringLenBetween(1, 255),
					validation.StringMatch(regexp.MustCompile(`^[a-zA-Z0-9_-]+$`), "must contain only alphanumeric characters, hyphens, and underscores"),
				),
			},
			"access_type": {
				Type:         schema.TypeString,
				Required:     true,
				Description:  "The access type for the bucket. Possible values are: `public`, `private`, `custom`. When set to `custom`, whitelist is required.",
				ValidateFunc: validation.StringInSlice([]string{"public", "private", "custom"}, false),
			},
			"whitelist": {
				Type:        schema.TypeList,
				Optional:    true,
				Description: "List of IP ranges in CIDR notation allowed to access the bucket. Required when access_type is `custom`.",
				Elem: &schema.Schema{
					Type: schema.TypeString,
				},
			},
			"versioning": {
				Type:         schema.TypeString,
				Optional:     true,
				Computed:     true,
				Description:  "The versioning status of the bucket. Possible values are: `Enabled`, `Suspended`. When read, may also be `Disabled` if versioning has never been enabled. Bucket has to be accessible from the Cloud Temple's Console to be able to configure this value.",
				ValidateFunc: validation.StringInSlice([]string{"Enabled", "Suspended"}, false),
			},
			"acl_entry": {
				Type:        schema.TypeSet,
				Optional:    true,
				Description: "ACL entries granting permissions to storage accounts on this bucket.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"storage_account": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The name of the storage account.",
						},
						"role": {
							Type:        schema.TypeString,
							Required:    true,
							Description: "The role to grant (e.g., 'READ', 'WRITE', 'FULL_CONTROL').",
						},
					},
				},
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The ID of the bucket.",
			},
			"namespace": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The namespace of the bucket.",
			},
			"retention_period": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The retention period of the bucket.",
			},
			"endpoint": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The endpoint URL of the bucket.",
			},
			"total_size": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The total size of objects in the bucket.",
			},
			"total_size_unit": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The unit for total_size.",
			},
			"total_objects": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The total number of objects in the bucket.",
			},
			"total_objects_deleted": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The total number of deleted objects.",
			},
			"total_size_deleted": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The total size of deleted objects.",
			},
			"tags": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The tags associated with the bucket.",
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

		CustomizeDiff: func(ctx context.Context, diff *schema.ResourceDiff, meta interface{}) error {
			accessType := diff.Get("access_type").(string)
			whitelist := diff.Get("whitelist").([]interface{})

			if accessType == "custom" && len(whitelist) == 0 {
				return fmt.Errorf("whitelist is required when access_type is 'custom'")
			}

			return nil
		},
	}
}

func objectStorageBucketCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	whitelist := []string{}
	for _, v := range d.Get("whitelist").([]interface{}) {
		whitelist = append(whitelist, v.(string))
	}

	activityId, err := c.ObjectStorage().Bucket().Create(ctx, &client.CreateBucketRequest{
		Name:       d.Get("name").(string),
		AccessType: d.Get("access_type").(string),
		Whitelist:  whitelist,
	})
	if err != nil {
		return diag.Errorf("failed to create bucket: %s", err)
	}

	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	setIdFromActivityState(d, activity)
	if err != nil {
		return diag.Errorf("failed to create bucket: %s", err)
	}

	// Configure versioning if specified
	if v, ok := d.GetOk("versioning"); ok {
		activityId, err := c.ObjectStorage().Bucket().UpdateVersioning(ctx, d.Get("name").(string), &client.UpdateVersioningRequest{
			Status: v.(string),
		})
		if err != nil {
			return diag.Errorf("failed to update versioning: %s", err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to update versioning: %s", err)
		}
	}

	// Grant ACL entries if specified
	if aclEntries, ok := d.GetOk("acl_entry"); ok {
		bucketName := d.Get("name").(string)
		for _, entry := range aclEntries.(*schema.Set).List() {
			entryMap := entry.(map[string]interface{})
			storageAccount := entryMap["storage_account"].(string)
			role := entryMap["role"].(string)

			activityId, err := c.ObjectStorage().ACLEntry().Grant(ctx, bucketName, role, storageAccount)
			if err != nil {
				return diag.Errorf("failed to grant ACL entry: %s", err)
			}
			if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
				return diag.Errorf("failed to grant ACL entry: %s", err)
			}
		}
	}

	return objectStorageBucketRead(ctx, d, meta)
}

func objectStorageBucketRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	bucket, err := c.ObjectStorage().Bucket().Read(ctx, d.Get("name").(string))
	if err != nil {
		return diag.Errorf("failed to read bucket: %s", err)
	}
	if bucket == nil {
		d.SetId("")
		return nil
	}

	// Use helper to flatten the bucket data
	bucketData := helpers.FlattenBucket(bucket)

	// Add tags
	tags := make([]map[string]interface{}, len(bucket.Tags))
	for i, tag := range bucket.Tags {
		tags[i] = map[string]interface{}{
			"key":   tag.Key,
			"value": tag.Value,
		}
	}
	bucketData["tags"] = tags

	// Set all data in state
	for k, v := range bucketData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	// Read ACL entries
	aclEntries, err := c.ObjectStorage().Bucket().ListACLEntries(ctx, d.Get("name").(string))
	if err != nil {
		return diag.Errorf("failed to read ACL entries: %s", err)
	}

	aclSet := schema.NewSet(schema.HashResource(&schema.Resource{
		Schema: map[string]*schema.Schema{
			"storage_account": {Type: schema.TypeString},
			"role":            {Type: schema.TypeString},
		},
	}), []interface{}{})

	for _, entry := range aclEntries {
		aclSet.Add(map[string]interface{}{
			"storage_account": entry.Name,
			"role":            entry.Role,
		})
	}

	if err := d.Set("acl_entry", aclSet); err != nil {
		return diag.FromErr(err)
	}

	return diags
}

func objectStorageBucketUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Update access_type or whitelist
	if d.HasChange("access_type") || d.HasChange("whitelist") {
		whitelist := []string{}
		for _, v := range d.Get("whitelist").([]interface{}) {
			whitelist = append(whitelist, v.(string))
		}

		activityId, err := c.ObjectStorage().Bucket().UpdateWhitelist(ctx, d.Get("name").(string), &client.UpdateWhitelistRequest{
			AccessType: d.Get("access_type").(string),
			Whitelist:  whitelist,
		})
		if err != nil {
			return diag.Errorf("failed to update bucket whitelist: %s", err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to update bucket whitelist: %s", err)
		}
	}

	// Update versioning
	if d.HasChange("versioning") {
		activityId, err := c.ObjectStorage().Bucket().UpdateVersioning(ctx, d.Get("name").(string), &client.UpdateVersioningRequest{
			Status: d.Get("versioning").(string),
		})
		if err != nil {
			return diag.Errorf("failed to update versioning: %s", err)
		}
		if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
			return diag.Errorf("failed to update versioning: %s", err)
		}
	}

	// Update ACL entries
	if d.HasChange("acl_entry") {
		bucketName := d.Get("name").(string)
		old, new := d.GetChange("acl_entry")
		oldSet := old.(*schema.Set)
		newSet := new.(*schema.Set)

		// Revoke removed entries
		toRevoke := oldSet.Difference(newSet)
		for _, entry := range toRevoke.List() {
			entryMap := entry.(map[string]interface{})
			storageAccount := entryMap["storage_account"].(string)
			role := entryMap["role"].(string)

			activityId, err := c.ObjectStorage().ACLEntry().Revoke(ctx, bucketName, role, storageAccount)
			if err != nil {
				return diag.Errorf("failed to revoke ACL entry: %s", err)
			}
			if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
				return diag.Errorf("failed to revoke ACL entry: %s", err)
			}
		}

		// Grant new entries
		toGrant := newSet.Difference(oldSet)
		for _, entry := range toGrant.List() {
			entryMap := entry.(map[string]interface{})
			storageAccount := entryMap["storage_account"].(string)
			role := entryMap["role"].(string)

			activityId, err := c.ObjectStorage().ACLEntry().Grant(ctx, bucketName, role, storageAccount)
			if err != nil {
				return diag.Errorf("failed to grant ACL entry: %s", err)
			}
			if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
				return diag.Errorf("failed to grant ACL entry: %s", err)
			}
		}
	}

	return objectStorageBucketRead(ctx, d, meta)
}

func objectStorageBucketDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.ObjectStorage().Bucket().Delete(ctx, d.Get("name").(string))
	if err != nil {
		return diag.Errorf("failed to delete bucket: %s", err)
	}
	if _, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx)); err != nil {
		return diag.Errorf("failed to delete bucket: %s", err)
	}

	return nil
}
