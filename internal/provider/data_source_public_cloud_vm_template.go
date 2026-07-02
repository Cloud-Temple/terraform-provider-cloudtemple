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

func dataSourcePublicCloudVMTemplate() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a Public Cloud VM Instances template (OS image), by `id` or by `name`.",

		ReadContext: publicCloudVMTemplateRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the template to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the template to retrieve (e.g. `Rocky Linux 9`). Conflicts with `id`.",
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
			"os_family":           {Type: schema.TypeString, Computed: true, Description: "The OS family of the template (e.g. `linux`)."},
			"os_name":             {Type: schema.TypeString, Computed: true, Description: "The OS name of the template (e.g. `Rocky Linux 9`)."},
			"os_version":          {Type: schema.TypeString, Computed: true, Description: "The OS version of the template."},
			"disk_sizes_gb":       {Type: schema.TypeList, Computed: true, Description: "The disk sizes (GB) provided by the template.", Elem: &schema.Schema{Type: schema.TypeInt}},
			"compatible_families": {Type: schema.TypeList, Computed: true, Description: "The IDs of the instance families this template is compatible with.", Elem: &schema.Schema{Type: schema.TypeString}},
			"categories":          {Type: schema.TypeList, Computed: true, Description: "The categories of the template.", Elem: &schema.Schema{Type: schema.TypeString}},
			"family":              {Type: schema.TypeString, Computed: true, Description: "The template family."},
			"version":             {Type: schema.TypeString, Computed: true, Description: "The template version."},
			"editor":              {Type: schema.TypeString, Computed: true, Description: "The template editor/publisher."},
			"description_en":      {Type: schema.TypeString, Computed: true, Description: "The English description of the template."},
			"template_type":       {Type: schema.TypeString, Computed: true, Description: "The template type."},
			"icon":                {Type: schema.TypeString, Computed: true, Description: "The template icon (data URI)."},
		},
	}
}

func publicCloudVMTemplateRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var tpl *client.PublicCloudVMTemplate

	if name := d.Get("name").(string); name != "" {
		tpls, err := c.PublicCloudVM().Template().List(ctx, &client.PublicCloudVMTemplateFilter{
			InstanceFamilyID:   d.Get("instance_family_id").(string),
			AvailabilityZoneID: d.Get("availability_zone_id").(string),
		})
		if err != nil {
			return diag.FromErr(fmt.Errorf("failed to find template named %q: %s", name, err))
		}
		// Names are not guaranteed unique: refuse an ambiguous match rather
		// than silently picking one.
		var matches []string
		for _, t := range tpls {
			if t.Name == name {
				tpl = t
				matches = append(matches, t.ID)
			}
		}
		if tpl == nil {
			return diag.FromErr(fmt.Errorf("failed to find template named %q", name))
		}
		if len(matches) > 1 {
			return diag.FromErr(fmt.Errorf("found %d templates named %q (ids: %s); narrow with instance_family_id and availability_zone_id, or use id", len(matches), name, strings.Join(matches, ", ")))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		var err error
		tpl, err = c.PublicCloudVM().Template().Read(ctx, id)
		if err != nil {
			return diag.FromErr(err)
		}
		if tpl == nil {
			return diag.FromErr(fmt.Errorf("failed to find template with id %q", id))
		}
	}

	d.SetId(tpl.ID)
	for k, v := range helpers.FlattenPublicCloudVMTemplate(tpl) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
