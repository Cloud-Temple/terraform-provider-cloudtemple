package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func resourceBackupSLAPolicyAssignment() *schema.Resource {
	return &schema.Resource{
		Description: "",

		CreateContext: computeBackupSLAPolicyAssignmentUpdate,
		ReadContext:   computeBackupSLAPolicyAssignmentRead,
		UpdateContext: computeBackupSLAPolicyAssignmentUpdate,
		DeleteContext: computeBackupSLAPolicyAssignmentDelete,

		Importer: &schema.ResourceImporter{
			StateContext: schema.ImportStatePassthroughContext,
		},

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ForceNew:     true,
				ValidateFunc: validation.IsUUID,
			},
			"sla_policy_ids": {
				Type:     schema.TypeSet,
				Required: true,

				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.IsUUID,
				},
			},
		},
	}
}

func computeBackupSLAPolicyAssignmentRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	virtualMachineId := d.Id()
	policies, err := c.Backup().SLAPolicy().List(ctx, &client.BackupSLAPolicyFilter{
		VirtualMachineId: virtualMachineId,
	})
	if err != nil {
		return diag.Errorf("failed to read SLA policies for virtual machine %q: %s", virtualMachineId, err)
	}

	res := []interface{}{}
	for _, policy := range policies {
		res = append(res, policy.ID)
	}
	sw := newStateWriter(d)
	sw.set("virtual_machine_id", virtualMachineId)
	sw.set("sla_policy_ids", res)

	return sw.diags
}

func computeBackupSLAPolicyAssignmentUpdate(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	// First we need to update the catalog
	jobs, err := c.Backup().Job().List(ctx, &client.BackupJobFilter{
		Type: "catalog",
	})
	if err != nil {
		return diag.Errorf("failed to find catalog job: %s", err)
	}

	activityId, err := c.Backup().Job().Run(ctx, &client.BackupJobRunRequest{
		JobId: jobs[0].ID,
	})
	if err != nil {
		return diag.Errorf("failed to update catalog: %s", err)
	}

	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update catalog, %s", err)
	}

	_, err = c.Backup().Job().WaitForCompletion(ctx, jobs[0].ID, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to update catalog, %s", err)
	}

	// Now we can assign the policies
	instanceId := d.Get("virtual_machine_id").(string)
	slaPolicies := []string{}
	for _, policy := range d.Get("sla_policy_ids").(*schema.Set).List() {
		slaPolicies = append(slaPolicies, policy.(string))
	}
	activityId, err = c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
		VirtualMachineIds: []string{instanceId},
		SLAPolicies:       slaPolicies,
	})
	if err != nil {
		return diag.Errorf("failed to assign SLA policies: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to assign SLA policies, %s", err)
	}

	d.SetId(instanceId)

	return computeBackupSLAPolicyAssignmentRead(ctx, d, meta)
}

func computeBackupSLAPolicyAssignmentDelete(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	instanceId := d.Get("virtual_machine_id").(string)
	activityId, err := c.Backup().SLAPolicy().AssignVirtualMachine(ctx, &client.BackupAssignVirtualMachineRequest{
		VirtualMachineIds: []string{instanceId},
		SLAPolicies:       []string{},
	})
	if err != nil {
		return diag.Errorf("failed to remove SLA policies: %s", err)
	}
	_, err = c.Activity().WaitForCompletion(ctx, activityId, getWaiterOptions(ctx))
	if err != nil {
		return diag.Errorf("failed to remove SLA policies, %s", err)
	}

	d.SetId(instanceId)
	return nil
}
