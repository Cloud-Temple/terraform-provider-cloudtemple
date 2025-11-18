package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBucketFiles() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve files from a bucket. Optionally filter by folder path. Bucket has to be accessible from the Cloud Temple's Console in order to get it's files listed.",

		ReadContext: dataSourceBucketFilesRead,

		Schema: map[string]*schema.Schema{
			// In
			"bucket_name": {
				Description: "The name of the bucket.",
				Type:        schema.TypeString,
				Required:    true,
			},
			"folder_path": {
				Description: "Optional folder path to filter files. If not specified, returns all files in the bucket.",
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
			},

			// Out
			"files": {
				Description: "The list of files in the bucket.",
				Type:        schema.TypeList,
				Computed:    true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"key": {
							Description: "The key/path of the file.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"last_modified": {
							Description: "The last modified date of the file.",
							Type:        schema.TypeString,
							Computed:    true,
						},
						"size": {
							Description: "The size of the file in bytes.",
							Type:        schema.TypeInt,
							Computed:    true,
						},
						"tags": {
							Description: "The tags associated with the file.",
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
						"versions": {
							Description: "The versions of the file.",
							Type:        schema.TypeList,
							Computed:    true,
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"version_id": {
										Description: "The version ID.",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"is_latest": {
										Description: "Whether this is the latest version.",
										Type:        schema.TypeBool,
										Computed:    true,
									},
									"last_modified": {
										Description: "The last modified date of the version.",
										Type:        schema.TypeString,
										Computed:    true,
									},
									"size": {
										Description: "The size of the version in bytes.",
										Type:        schema.TypeInt,
										Computed:    true,
									},
									"is_delete_marker": {
										Description: "Whether this version is a delete marker.",
										Type:        schema.TypeBool,
										Computed:    true,
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

// dataSourceBucketFilesRead lit les fichiers d'un bucket et les mappe dans le state Terraform
func dataSourceBucketFilesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	bucketName := d.Get("bucket_name").(string)
	folderPath := d.Get("folder_path").(string)

	// Récupérer les fichiers du bucket
	files, err := c.ObjectStorage().BucketFiles().List(ctx, bucketName, &client.BucketFilesFilter{
		FolderPath: folderPath,
	})
	if err != nil {
		return diag.FromErr(fmt.Errorf("error listing files for bucket %s: %s", bucketName, err))
	}

	// Définir l'ID de la datasource
	if folderPath != "" {
		d.SetId(fmt.Sprintf("%s/%s", bucketName, folderPath))
	} else {
		d.SetId(bucketName)
	}

	// Mapper les données en utilisant la fonction helper
	tfFiles := make([]map[string]interface{}, len(files))
	for i, file := range files {
		tfFiles[i] = helpers.FlattenBucketFile(file)
	}

	// Définir les données dans le state
	if err := d.Set("files", tfFiles); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
