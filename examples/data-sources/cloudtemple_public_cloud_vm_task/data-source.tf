# Retrieve a diagnostic task by id.
# NOTE: tasks are diagnostic only and are unrelated to the activities that track
# writes; never use a task to follow a resource create/update/delete.
data "cloudtemple_public_cloud_vm_task" "t" {
  id = "441e1d65-6639-497b-b41b-540955b83a03"
}

output "task_status" {
  value = data.cloudtemple_public_cloud_vm_task.t.status
}
