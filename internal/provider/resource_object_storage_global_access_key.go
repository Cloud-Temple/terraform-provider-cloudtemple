package provider

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceGlobalAccessKey() *schema.Resource {
	return &schema.Resource{
		Description: "Manage the global access key for object storage. This is a singleton resource that renews the access key credentials.",

		CreateContext: objectStorageGlobalAccessKeyCreate,
		ReadContext:   objectStorageGlobalAccessKeyRead,
		DeleteContext: objectStorageGlobalAccessKeyDelete,

		Schema: map[string]*schema.Schema{
			// Out
			"access_key_id": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The access key ID.",
			},
			"access_secret_key": {
				Type:        schema.TypeString,
				Computed:    true,
				Sensitive:   true,
				Description: "The secret access key. This is only available after renewal (creation).",
			},
		},
	}
}

func objectStorageGlobalAccessKeyCreate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Renew the global access key
	renewResp, err := c.ObjectStorage().GlobalAccessKey().Renew(ctx)
	if err != nil {
		return diag.Errorf("failed to renew global access key: %s", err)
	}

	// Set the singleton ID
	d.SetId("global_access_key")

	// Set the sensitive credentials
	d.Set("access_key_id", renewResp.AccessKeyID)
	d.Set("access_secret_key", renewResp.AccessSecretKey)

	return objectStorageGlobalAccessKeyRead(ctx, d, meta)
}

func objectStorageGlobalAccessKeyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// Get namespace information to verify the key still exists
	namespace, err := c.ObjectStorage().Namespace().Read(ctx)
	if err != nil {
		return diag.Errorf("failed to read namespace: %s", err)
	}

	// Set access_key_id from namespace (secret is only available at creation)
	d.Set("access_key_id", namespace.AccessKeyID)

	return nil
}

func objectStorageGlobalAccessKeyDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	// No actual deletion - just remove from state
	// The global access key cannot be deleted, only renewed
	return nil
}
