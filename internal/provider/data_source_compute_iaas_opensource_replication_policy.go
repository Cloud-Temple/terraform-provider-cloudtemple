package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourceOpenIaasReplicationPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a specific replication policy from an Open IaaS infrastructure.",

		ReadContext: computeOpenIaaSReplicationPolicyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the replication policy to retrieve.",
			},

			// Out
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
	}
}

func computeOpenIaaSReplicationPolicyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	id := d.Get("id").(string)
	policy, err := c.Compute().OpenIaaS().Replication().Policy().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if policy == nil {
		return diag.FromErr(fmt.Errorf("failed to find replication policy with id %q", id))
	}

	// Refuse a timestamp the state cannot hold exactly BEFORE anything is
	// written, so a refused read leaves the data source untouched (#543).
	if err := helpers.ValidateOpenIaaSReplicationPolicyLastRun(policy); err != nil {
		return diag.FromErr(err)
	}

	d.SetId(policy.ID)

	policyData := helpers.FlattenOpenIaaSReplicationPolicy(policy)
	for k, v := range policyData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}
