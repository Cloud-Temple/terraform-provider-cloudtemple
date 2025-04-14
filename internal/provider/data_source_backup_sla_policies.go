package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceBackupSLAPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "",

		ReadContext: backupSLAPoliciesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"virtual_machine_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},
			"virtual_disk_id": {
				Type:     schema.TypeString,
				Optional: true,
				Default:  "",
			},

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

// backupSLAPoliciesRead lit les politiques SLA de backup et les mappe dans le state Terraform
func backupSLAPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	var c *client.Client = getClient(meta)
	var diags diag.Diagnostics

	// Récupérer les politiques SLA
	slaPolicies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{
		VirtualMachineId: d.Get("virtual_machine_id").(string),
		VirtualDiskId:    d.Get("virtual_disk_id").(string),
	})
	if err != nil {
		return diag.FromErr(err)
	}

	// Définir l'ID de la datasource
	d.SetId("sla_policies")

	// Mapper manuellement les données en utilisant la fonction helper
	tfSLAPolicies := make([]map[string]interface{}, len(slaPolicies))
	for i, policy := range slaPolicies {
		tfSLAPolicies[i] = helpers.FlattenBackupSLAPolicy(policy)
	}

	// Définir les données dans le state
	if err := d.Set("sla_policies", tfSLAPolicies); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
