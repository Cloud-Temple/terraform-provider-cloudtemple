package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBuckets() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all buckets in the object storage.",

		ReadContext: dataSourceBucketsRead,

		Schema: map[string]*schema.Schema{
			// Out
			"buckets": {
				Description: "The list of buckets.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Description: "The ID of the bucket.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"name": {
							Description: "The name of the bucket.",
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
				},
			},
		},
	}
}

// dataSourceBucketsRead lit les buckets et les mappe dans le state Terraform
func dataSourceBucketsRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer tous les buckets
	buckets, err := c.ObjectStorage().Bucket().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("buckets")

	// Mapper les données en utilisant la fonction helper
	tfBuckets := make([]map[string]interface{}, len(buckets))
	for i, bucket := range buckets {
		tfBuckets[i] = helpers.FlattenBucket(bucket)
	}

	// Définir les données dans le state
	if err := d.Set("buckets", tfBuckets); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
