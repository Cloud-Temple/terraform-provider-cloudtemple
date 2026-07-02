# Retrieve a diagnostic task by id. Tasks are upstream diagnostic objects — the
# provider tracks its own writes through the Activities service, never through
# tasks.
data "cloudtemple_public_cloud_vm_task" "t" {
  id = "00000000-0000-0000-0000-000000000000"
}

output "task_status" {
  value = data.cloudtemple_public_cloud_vm_task.t.status
}
