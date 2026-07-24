package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMImage() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances OS image, by `id` or by `name`.",

		ReadContext: publicCloudVMImageRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the image to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the image to retrieve (e.g. `Rocky Linux 9`). Conflicts with `id`.",
			},
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Instance family ID, used to narrow the search when looking up by `name`.",
			},
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Availability zone ID, used to narrow the search when looking up by `name`.",
			},

			// Out
			"os_family":           {Type: schema.TypeString, Computed: true, Description: "The OS family of the image (e.g. `linux`)."},
			"os_name":             {Type: schema.TypeString, Computed: true, Description: "The OS name of the image (e.g. `Rocky Linux 9`)."},
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
	}
}

func publicCloudVMImageRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var img *client.PublicCloudVMImage

	if name := d.Get("name").(string); name != "" {
		imgs, err := c.PublicCloudVM().Image().List(ctx, &client.PublicCloudVMImageFilter{
			InstanceFamilyID:   d.Get("instance_family_id").(string),
			AvailabilityZoneID: d.Get("availability_zone_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find image named %q: %s", name, err))
		}
		// Names are not guaranteed unique: refuse an ambiguous match rather
		// than silently picking one.
		var matches []string
		for _, i := range imgs {
			if i.Name == name {
				img = i
				matches = append(matches, i.ID)
			}
		}
		if img == nil {
			return diag.FromErr(fmt.Errorf("failed to find image named %q", name))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d images named %q (ids: %s); narrow with instance_family_id and availability_zone_id, or use id", len(matches), name, strings.Join(matches, ", ")))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		img, err = c.PublicCloudVM().Image().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if img == nil {
			return diag.FromErr(fmt.Errorf("failed to find image with id %q", id))
		}
	}

	d.SetId(img.ID)
	for k, v := range helpers.FlattenPublicCloudVMImage(img) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
