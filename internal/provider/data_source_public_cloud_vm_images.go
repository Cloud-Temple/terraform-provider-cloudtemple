package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMImages() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances OS images of the tenant.",

		ReadContext: publicCloudVMImagesRead,

		Schema: map[string]*schema.Schema{
			// In
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter images by instance family ID.",
			},
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter images by availability zone ID.",
			},

			// Out
			"images": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of images.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                  {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the image."},
						"name":                {Type: schema.TypeString, Computed: true, Description: "The name of the image (e.g. `Rocky Linux 9`)."},
						"os_family":           {Type: schema.TypeString, Computed: true, Description: "The OS family of the image (e.g. `linux`)."},
						"os_name":             {Type: schema.TypeString, Computed: true, Description: "The OS name of the image."},
						"os_version":          {Type: schema.TypeString, Computed: true, Description: "The OS version of the image."},
						"disk_sizes_gb":       {Type: schema.TypeList, Computed: true, Description: "The disk sizes (GB) provided by the image.", Elem: &schema.Schema{Type: schema.TypeInt}},
						"compatible_families": {Type: schema.TypeList, Computed: true, Description: "The IDs of the instance families this image is compatible with.", Elem: &schema.Schema{Type: schema.TypeString}},
						"categories":          {Type: schema.TypeList, Computed: true, Description: "The categories of the image.", Elem: &schema.Schema{Type: schema.TypeString}},
						"family":              {Type: schema.TypeString, Computed: true, Description: "The image family."},
						"version":             {Type: schema.TypeString, Computed: true, Description: "The image version."},
						"editor":              {Type: schema.TypeString, Computed: true, Description: "The image editor/publisher."},
						"description_en":      {Type: schema.TypeString, Computed: true, Description: "The English description of the image."},
						"image_type":          {Type: schema.TypeString, Computed: true, Description: "The image type."},
						"icon":                {Type: schema.TypeString, Computed: true, Description: "The image icon (data URI)."},
					},
				},
			},
		},
	}
}

func publicCloudVMImagesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	imgs, err := c.PublicCloudVM().Image().List(ctx, &client.PublicCloudVMImageFilter{
		InstanceFamilyID:   d.Get("instance_family_id").(string),
		AvailabilityZoneID: d.Get("availability_zone_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_images")

	tfImages := make([]map[string]interface{}, len(imgs))
	for i, img := range imgs {
		tfImages[i] = helpers.FlattenPublicCloudVMImage(img)
	}

	if err := d.Set("images", tfImages); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
