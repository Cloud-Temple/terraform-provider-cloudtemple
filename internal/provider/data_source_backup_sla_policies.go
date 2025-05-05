package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceBackupSLAPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a list of backup SLA policies.",

		ReadContext: backupSLAPoliciesRead,

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Optional:    true,
				Default:     "",
				Description: "Filter policies by name.",
			},
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter policies by virtual machine ID.",
			},
			"virtual_disk_id": {
				Type:         schema.TypeString,
				Optional:     true,
				Default:      "",
				ValidateFunc: validation.IsUUID,
				Description:  "Filter policies by virtual disk ID.",
			},

			// Out
			"sla_policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of SLA policies matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the SLA policy.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the SLA policy.",
						},
						"sub_policies": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "List of sub-policies contained within this SLA policy.",

							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"type": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The type of the sub-policy.",
									},
									"use_encryption": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether encryption is used for this sub-policy.",
									},
									"software": {
										Type:        schema.TypeBool,
										Computed:    true,
										Description: "Indicates whether this is a software-based sub-policy.",
									},
									"site": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The site associated with this sub-policy.",
									},
									"retention": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Retention settings for this sub-policy.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"age": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The retention age in days for backups created by this sub-policy.",
												},
											},
										},
									},
									"trigger": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Trigger settings for this sub-policy.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"frequency": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The frequency of the trigger.",
												},
												"type": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The type of the trigger. (eg. SUBHOURLY, HOURLY, DAILY, WEEKLY, MONTHLY)",
												},
												"activate_date": {
													Type:        schema.TypeInt,
													Computed:    true,
													Description: "The activation date of the trigger as a Unix timestamp.",
												},
											},
										},
									},
									"target": {
										Type:        schema.TypeList,
										Computed:    true,
										Description: "Target settings for this sub-policy.",

										Elem: &schema.Resource{
											Schema: map[string]*schema.Schema{
												"id": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The ID of the target resource.",
												},
												"href": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The href (URL) of the target resource.",
												},
												"resource_type": {
													Type:        schema.TypeString,
													Computed:    true,
													Description: "The type of the target resource.",
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
