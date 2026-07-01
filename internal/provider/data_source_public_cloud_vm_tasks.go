package provider

import (
	"context"

	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/provider/helpers"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func dataSourcePublicCloudVMTasks() *schema.Resource {
	return &schema.Resource{
		Description: "Used to retrieve Public Cloud VM Instances tasks (DIAGNOSTIC objects). Optionally scoped to a VM. Tasks are unrelated to the activities that track writes and must never be used to follow a write. This is a diagnostic listing and is not auto-paginated.",

		ReadContext: publicCloudVMTasksRead,

		Schema: map[string]*schema.Schema{
			// In
			"virtual_machine_id": {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.IsUUID,
				Description:  "If set, only the tasks of this VM are returned (uses the per-VM endpoint).",
			},
			"limit": {
				Type:        schema.TypeInt,
				Optional:    true,
				Description: "Maximum number of tasks to return (API max: 500 global / 200 per-VM). 0 uses the API default.",
			},

			// Out
			"tasks": {
				Type:        schema.TypeList,
				Computed:    true,
				Description: "The list of tasks.",
				Elem: &schema.Resource{
					Schema: map[string]*schema.Schema{
						"id":              {Type: schema.TypeString, Computed: true, Description: "The unique identifier of the task."},
						"vm_id":           {Type: schema.TypeString, Computed: true, Description: "The ID of the VM the task relates to (may be empty)."},
						"task_type":       {Type: schema.TypeString, Computed: true, Description: "The type of the task."},
						"status":          {Type: schema.TypeString, Computed: true, Description: "The status of the task."},
						"message":         {Type: schema.TypeString, Computed: true, Description: "A human-readable message describing the task outcome."},
						"failure_code":    {Type: schema.TypeString, Computed: true, Description: "A stable failure code when the task failed."},
						"retried_from_id": {Type: schema.TypeString, Computed: true, Description: "The ID of the task this task was retried from, if any."},
						"created_at":      {Type: schema.TypeString, Computed: true, Description: "The creation date of the task (RFC3339)."},
						"updated_at":      {Type: schema.TypeString, Computed: true, Description: "The last update date of the task (RFC3339)."},
						"completed_at":    {Type: schema.TypeString, Computed: true, Description: "The completion date of the task (RFC3339), if completed."},
					},
				},
			},
		},
	}
}

func publicCloudVMTasksRead(ctx context.Context, d *schema.ResourceData, meta any) diag.Diagnostics {
	c := getClient(meta)

	tasks, err := c.PublicCloudVM().Task().List(ctx, d.Get("virtual_machine_id").(string), d.Get("limit").(int))
	if err != nil {
		return diag.FromErr(err)
	}

	d.SetId("public_cloud_vm_tasks")

	tfTasks := make([]map[string]interface{}, len(tasks))
	for i, task := range tasks {
		tfTasks[i] = helpers.FlattenPublicCloudVMTask(task)
	}

	if err := d.Set("tasks", tfTasks); err != nil {
		return diag.FromErr(err)
	}

	return nil
}
