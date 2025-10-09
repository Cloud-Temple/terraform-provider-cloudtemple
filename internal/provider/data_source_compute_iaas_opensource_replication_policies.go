package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func dataSourceOpenIaasReplicationPolicies() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve all replication policies from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSReplicationPoliciesRead,

		Schema: map[string]*schema.Schema{
			// In

			// Out
			"policies": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "List of replication policies matching the filter criteria.",

				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The unique identifier of the replication policy.",
						},
						"name": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The name of the replication policy.",
						},
						"machine_manager_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the machine manager (availability zone).",
						},
						"storage_repository_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the storage repository.",
						},
						"pool_id": {
							Type:        schema.TypeString,
							Computed:    true,
							Description: "The ID of the pool.",
						},
						"interval": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "The replication interval configuration.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"hours": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of hours in the replication interval.",
									},
									"minutes": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The number of minutes in the replication interval.",
									},
								},
							},
						},
						"last_run": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the last replication run.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"start": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The start timestamp of the last replication run.",
									},
									"end": {
										Type:        schema.TypeInt,
										Computed:    true,
										Description: "The end timestamp of the last replication run.",
									},
									"status": {
										Type:        schema.TypeString,
										Computed:    true,
										Description: "The status of the last replication run.",
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

func computeOpenIaaSReplicationPoliciesRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	policies, err := c.Compute().OpenIaaS().Replication().Policy().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("openiaas_replication_policies")

	tfPolicies := make([]map[string]interface{}, len(policies))
	for i, policy := range policies {
		tfPolicies[i] = helpers.FlattenOpenIaaSReplicationPolicy(policy)
	}

	if err := d.Set("policies", tfPolicies); err != nil {
		return diag.FromErr(err)
	}

	return diags
}
