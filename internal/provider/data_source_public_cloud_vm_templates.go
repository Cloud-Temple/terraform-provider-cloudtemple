package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMTemplates() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all Public Cloud VM Instances templates (OS images) of the tenant.",

		ReadContext: publicCloudVMTemplatesRead,

		Schema: map[string]*schema.Schema{
			// In
			"instance_family_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter templates by instance family ID.",
			},
			"availability_zone_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "Filter templates by availability zone ID.",
			},

			// Out
			"templates": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of templates.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":                  {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the template."},
						"name":                {Type: schema.TypeString, Computed: true, Description: "The name of the template (e.g. `Rocky Linux 9`)."},
						"os_family":           {Type: schema.TypeString, Computed: true, Description: "The OS family of the template (e.g. `linux`)."},
						"os_name":             {Type: schema.TypeString, Computed: true, Description: "The OS name of the template."},
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
				},
			},
		},
	}
}

func publicCloudVMTemplatesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	tpls, err := c.PublicCloudVM().Template().List(ctx, &client.PublicCloudVMTemplateFilter{
		InstanceFamilyID:   d.Get("instance_family_id").(string),
		AvailabilityZoneID: d.Get("availability_zone_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_templates")

	tfTemplates := make([]map[string]interface{}, len(tpls))
	for i, t := range tpls {
		tfTemplates[i] = helpers.FlattenPublicCloudVMTemplate(t)
	}

	if err := d.Set("templates", tfTemplates); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
