variable "virtual_machine_id" {
  type        = string
  description = "Scope the diagnostic tasks to this VM."
}

# List the diagnostic tasks of a VM (e.g. to investigate a failed operation).
data "cloudtemple_public_cloud_vm_tasks" "vm" {
  virtual_machine_id = var.virtual_machine_id
}

output "failed_tasks" {
  value = [
    for t in data.cloudtemple_public_cloud_vm_tasks.vm.tasks :
    "${t.task_type}: ${t.message}" if t.status == "failed"
  ]
}
