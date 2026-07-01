package provider

import (
	"context"
	"fmt"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMTask() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve a single Public Cloud VM Instances task by `id`. Tasks are a DIAGNOSTIC object (upstream machine-manager); they are unrelated to the activities that track writes and must never be used to follow a write.",

		ReadContext: publicCloudVMTaskRead,

		Schema: map[string]*schema.Schema{
			// In
			"id": {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "The ID of the task to retrieve. (The live API returns UUID task ids; the underlying schema documents an opaque string.)",
			},

			// Out
			"vm_id":           {Type: schema.TypeString, Computed: true, Description: "The ID of the VM the task relates to (may be empty)."},
			"task_type":       {Type: schema.TypeString, Computed: true, Description: "The type of the task."},
			"status":          {Type: schema.TypeString, Computed: true, Description: "The status of the task (pending, running, completed, failed, cancelled...)."},
			"message":         {Type: schema.TypeString, Computed: true, Description: "A human-readable message describing the task outcome."},
			"failure_code":    {Type: schema.TypeString, Computed: true, Description: "A stable failure code when the task failed (e.g. InsufficientStorageCapacity)."},
			"retried_from_id": {Type: schema.TypeString, Computed: true, Description: "The ID of the task this task was retried from, if any."},
			"created_at":      {Type: schema.TypeString, Computed: true, Description: "The creation date of the task (RFC3339)."},
			"updated_at":      {Type: schema.TypeString, Computed: true, Description: "The last update date of the task (RFC3339)."},
			"completed_at":    {Type: schema.TypeString, Computed: true, Description: "The completion date of the task (RFC3339), if completed."},
		},
	}
}

func publicCloudVMTaskRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	id := d.Get("id").(string)
	task, err := c.PublicCloudVM().Task().Read(ctx, id)
	if err != nil {
		return diag.FromErr(err)
	}
	if task == nil {
		return diag.FromErr(fmt.Errorf("failed to find task with id %q", id))
	}

	d.SetId(task.ID)
	for k, v := range helpers.FlattenPublicCloudVMTask(task) {
		if err := d.Set(k, v); err != nil {
			return diag.FromErr(err)
		}
	}

	return nil
}
