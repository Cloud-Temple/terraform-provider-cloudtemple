package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMBackupPolicy() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a single Public Cloud VM Instances backup policy, by `id` or by `name`. A backup policy is required to create a VM (`backup_policy_id`). The API has no by-id endpoint; selection is done by listing and matching.",

		ReadContext: publicCloudVMBackupPolicyRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"name"},
				AtLeastOneOf:  []string{"id", "name"},
				ValidateFunc:  validation.IsUUID,
				Description:   "The ID of the backup policy to retrieve. Conflicts with `name`.",
			},
			"name": {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{"id"},
				AtLeastOneOf:  []string{"id", "name"},
				Description:   "The name of the backup policy to retrieve. Conflicts with `id`.",
			},

			// Out
			"description": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The human-readable description of the backup policy.",
			},
			"retention": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The number of restore points retained by the policy.",
			},
			"schedule_cron": {
				Type:        schema.TypeString,
				Computed:    true,
				Description: "The cron expression driving the backup schedule.",
			},
			"schedule_window_start_hour": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The hour of day (0-23) at which the backup window starts.",
			},
			"schedule_window_duration_minutes": {
				Type:        schema.TypeInt,
				Computed:    true,
				Description: "The duration (minutes) of the backup window.",
			},
		},
	}
}

func publicCloudVMBackupPolicyRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	policies, err := c.PublicCloudVM().BackupPolicy().List(ctx)
	if err != nil {
		return diag.FromErr(err)
	}

	var policy *client.PublicCloudVMBackupPolicy
	if name := d.Get("name").(string); name != "" {
		for _, p := range policies {
			if p.Name == name {
				policy = p
				break
			}
		}
		if policy == nil {
			return diag.FromErr(fmt.Errorf("failed to find backup policy named %q", name))
		}
	} else {
		id := d.Get("id").(string)
		if id == "" {
			return diag.FromErr(fmt.Errorf("either id or name must be specified"))
		}
		for _, p := range policies {
			if p.ID == id {
				policy = p
				break
			}
		}
		if policy == nil {
			return diag.FromErr(fmt.Errorf("failed to find backup policy with id %q", id))
		}
	}

	d.SetId(policy.ID)
	for k, v := range helpers.FlattenPublicCloudVMBackupPolicy(policy) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
