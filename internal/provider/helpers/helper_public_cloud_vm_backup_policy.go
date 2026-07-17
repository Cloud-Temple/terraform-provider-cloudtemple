package helpers

import (
	"github.com/cloud-temple/terraform-provider-cloudtemple/internal/client"
)

// FlattenPublicCloudVMBackupPolicy maps a client.PublicCloudVMBackupPolicy to the
// flat snake_case map consumed by both the single and list datasources.
func FlattenPublicCloudVMBackupPolicy(policy *client.PublicCloudVMBackupPolicy) map[string]interface{} {
	return map[string]interface{}{
		"id":                               policy.ID,
		"name":                             policy.Name,
		"description":                      policy.Description,
		"retention":                        policy.Retention,
		"schedule_cron":                    policy.ScheduleCron,
		"schedule_window_start_hour":       policy.ScheduleWindowStartHour,
		"schedule_window_duration_minutes": policy.ScheduleWindowDurationMinutes,
	}
}
