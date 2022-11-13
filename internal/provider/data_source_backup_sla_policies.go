package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupSLAPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: readFullResource(func(ctx context.Context, client *client.Client, d *schema.ResourceData) (interface{}, error) {
			slaPolicies, err := client.Backup().SLAPolicy().List(ctx, nil)
			return map[string]interface{}{
				"id":           "sla_policies",
				"sla_policies": slaPolicies,
			}, err
		}),

		Schema: map[string]*schema.Schema{
			// Out
			"sla_policies": {
				Type:     schema.TypeList,
				Computed: true,

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"name": {
							Type:     schema.TypeString,
							Computed: true,
						},
						"sub_policies": {
							Type:     schema.TypeList,
							Computed: true,

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"use_encryption": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"software": {
										Type:     schema.TypeBool,
										Computed: true,
									},
									"site": {
										Type:     schema.TypeString,
										Computed: true,
									},
									"retention": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"age": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"trigger": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"frequency": {
													Type:     schema.TypeInt,
													Computed: true,
												},
												"type": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"activate_date": {
													Type:     schema.TypeInt,
													Computed: true,
												},
											},
										},
									},
									"target": {
										Type:     schema.TypeList,
										Computed: true,

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"href": {
													Type:     schema.TypeString,
													Computed: true,
												},
												"resource_type": {
													Type:     schema.TypeString,
													Computed: true,
												},
											},
										},
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
