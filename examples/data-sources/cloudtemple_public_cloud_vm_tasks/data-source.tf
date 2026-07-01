# Retrieve recent diagnostic tasks (optionally scoped to a VM).
data "cloudtemple_public_cloud_vm_tasks" "recent" {
  limit = 20
}

# Or only the tasks of a given VM.
data "cloudtemple_public_cloud_vm_tasks" "for_vm" {
  virtual_machine_id = "e2f77153-21a9-4147-9638-6fba09a97b0e"
}

output "recent_task_types" {
  value = [for t in data.cloudtemple_public_cloud_vm_tasks.recent.tasks : t.task_type]
}
