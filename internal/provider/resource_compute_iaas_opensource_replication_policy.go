package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func resourceOpenIaasReplicationPolicy() *schema.Resource {
	return &schema.Resource{
		CreateContext: openIaasReplicationPolicyCreate,
		ReadContext:   openIaasReplicationPolicyRead,
		DeleteContext: openIaasReplicationPolicyDelete,
		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"name": {
				Type:        schema.TypeString,
				Description: "The name of the replication policy.",
				ForceNew:    true,
				Required:    true,
			},
			"storage_repository_id": {
				Type:        schema.TypeString,
				Description: "The ID of the storage repository where the replication policy is applied.",
				Required:    true,
				ForceNew:    true,
			},
			"interval": {
				Type:        schema.TypeList,
				Description: "The interval at which the replication policy runs.",
				Required:    true,
				ForceNew:    true,
				MaxItems:    1,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"hours": {
							Type:          schema.TypeInt,
							Description:   "The interval in hours.",
							ConflictsWith: []string{"interval.0.minutes"},
							Optional:      true,
						},
						"minutes": {
							Type:          schema.TypeInt,
							Description:   "The interval in minutes.",
							ConflictsWith: []string{"interval.0.hours"},
							Optional:      true,
						},
					},
				},
			},

			// Out
			"id": {
				Type:        schema.TypeString,
				Description: "The ID of the replication policy.",
				Computed:    true,
			},
			"pool_id": {
				Type:        schema.TypeString,
				Description: "The ID of the pool associated with the replication policy.",
				Computed:    true,
			},
			"machine_manager_id": {
				Type:        schema.TypeString,
				Description: "The ID of the machine manager associated with the replication policy.",
				Computed:    true,
			},
			"last_run": {
				Type:        schema.TypeList,
				Description: "The timestamp of the last run of the replication policy.",
				Computed:    true,
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"start": {
							Type:        schema.TypeInt,
							Description: "The start time of the last run as a timestamp.",
							Optional:    true,
						},
						"end": {
							Type:        schema.TypeInt,
							Description: "The end time of the last run as a timestamp.",
							Optional:    true,
						},
						"status": {
							Type:        schema.TypeString,
							Description: "The status of the last run.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func openIaasReplicationPolicyCreate(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)

	activityId, err := c.Compute().OpenIaaS().Replication().Policy().Create(ctx, &client.CreateOpenIaaSReplicationPolicyRequest{
		Name:                d.Get("name").(string),
		StorageRepositoryID: d.Get("storage_repository_id").(string),
		Interval: client.ReplicationPolicyInterval{
			Hours:   d.Get("interval").([]any)[0].(map[string]any)["hours"].(int),
			Minutes: d.Get("interval").([]any)[0].(map[string]any)["minutes"].(int),
		},
	})
	if err != nil {
		return diag.Errorf("the replication policy could not be created: %s", err)
	}
	activity, err := c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions((ctx)))
	setIdFromActivityState(d, activity)
	if err != nil {
		return diag.Errorf("the replication policy could not be created: %s", err)
	}

	return openIaasReplicationPolicyRead(ctx, d, meta)
}

func openIaasReplicationPolicyRead(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	replicationPolicy, err := c.Compute().OpenIaaS().Replication().Policy().Read(ctx, d.Id())
	if replicationPolicy == nil || err != nil {
		d.SetId("")
		return nil
	}

	policyData := helpers.FlattenOpenIaaSReplicationPolicy(replicationPolicy)

	for k, v := range policyData {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return diags
}

func openIaasReplicationPolicyDelete(ctx context.Context, d *schema.ResourceData, meta interface{}) diag.Diagnostics {
	c := getClient(meta)
	var diags diag.Diagnostics

	activityId, err := c.Compute().OpenIaaS().Replication().Policy().Delete(ctx, d.Id())
	if err != nil {
		return diag.FromErr(err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions((ctx)))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("")

	return diags
}
