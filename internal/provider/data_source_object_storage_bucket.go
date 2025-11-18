package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBucket() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve information about a specific bucket.",

		ReadContext: dataSourceBucketRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Description: "The name of the bucket.",
				Type:        schema.TypeString,
				Required:    true,
			},

			// Out
			"id": {
				Description: "The ID of the bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"namespace": {
				Description: "The namespace of the bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"retention_period": {
				Description: "The retention period of the bucket in days.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"versioning": {
				Description: "The versioning status of the bucket. (Value can be 'Enabled', 'Suspended' or 'Disabled')",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"endpoint": {
				Description: "The endpoint URL of the bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"total_size": {
				Description: "The total size of the bucket in bytes.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"total_size_unit": {
				Description: "The unit of the total size of the bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"total_objects": {
				Description: "The total number of objects in the bucket.",
				Type:        schema.TypeInt,
				Computed:    true,
			},
			"tags": {
				Description: "The tags associated with the bucket.",
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
			"total_objects_deleted": {
				Description: "The total number of deleted objects in the bucket.",
				Type:        schema.TypeString,
				Computed:    true,
			},
			"total_size_deleted": {
				Description: "The total size of deleted objects in the bucket in bytes.",
				Type:        schema.TypeString,
				Computed:    true,
			},
		},
	}
}

// dataSourceBucketRead lit un utilisateur et le mappe dans le state Terraform
func dataSourceBucketRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics
	var bucket *client.Bucket
	var err error

	bucketName := d.Get("name").(string)

	// Récupérer le bucket par son nom
	bucket, err = c.ObjectStorage().Bucket().Read(ctx, bucketName)
	if err != nil {
		return diag.FromErr(fmt.Errorf("error reading bucket with name %s: %s", bucketName, err))
	}

	// Définir l'ID de la datasource
	d.SetId(bucket.ID)

	// Mapper les données en utilisant la fonction helper
	bucketData := helpers.FlattenBucket(bucket)

	// Définir les données dans le state
	for k, v := range bucketData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
