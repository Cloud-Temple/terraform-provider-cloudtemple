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
						// 32-bit safety (#543): `start` and `end` carry Unix timestamps in
						// MILLISECONDS, which overflow a 32-bit int. They are declared TypeFloat, NOT
						// TypeInt, because the SDK's TypeInt writer decodes into a native `int`
						// (helper/schema/field_writer_map.go: `var n int; mapstructure.Decode(v, &n)`)
						// and mapstructure truncates instead of range-checking, so on the 386/arm
						// builds a timestamp would be written to the state silently wrong (measured:
						// 1759830000000 -> -1106591360). TypeFloat decodes into a float64 on every
						// platform, and both TypeInt and TypeFloat map to the SAME cty.Number at the
						// protocol level (helper/schema/core_schema.go), so the published contract is
						// unchanged and no state migration is needed. The millisecond precision of the
						// stored value is guaranteed up to 2^53 ms and enforced before the write by
						// helpers.ValidateOpenIaaSReplicationPolicyLastRun.
						"last_run": {
							Type:        schema.TypeList,
							Computed:    true,
							Description: "Information about the last replication run.",
							Elem: &schema.Resource{
								Schema: map[string]*schema.Schema{
									"start": {
										Type:        schema.TypeFloat,
										Computed:    true,
										Description: "The start timestamp of the last replication run.",
									},
									"end": {
										Type:        schema.TypeFloat,
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

	// Refuse a timestamp the state cannot hold exactly BEFORE anything is
	// written, so a refused read leaves the data source untouched (#543). The
	// whole listing is refused rather than the offending policy: silently
	// dropping it would shrink the listing with no evidence the policy is gone,
	// and keeping it would record a rounded timestamp — both are exactly what
	// the state-safety doctrine forbids.
	for _, policy := range policies {
		if err := helpers.ValidateOpenIaaSReplicationPolicyLastRun(policy); err != nil {
			return diag.FromErr(err)
		}
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
