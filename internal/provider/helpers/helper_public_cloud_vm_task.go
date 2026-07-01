package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMTask maps a client.PublicCloudVMTask to the flat snake_case
// map consumed by both the single and list datasources.
func FlattenPublicCloudVMTask(task *client.PublicCloudVMTask) map[string]interface{} {
	return map[string]interface{}{
		"id":              task.ID,
		"vm_id":           task.VmID,
		"task_type":       task.TaskType,
		"status":          task.Status,
		"message":         task.Message,
		"failure_code":    task.FailureCode,
		"retried_from_id": task.RetriedFromID,
		"created_at":      task.CreatedAt,
		"updated_at":      task.UpdatedAt,
		"completed_at":    task.CompletedAt,
	}
}
