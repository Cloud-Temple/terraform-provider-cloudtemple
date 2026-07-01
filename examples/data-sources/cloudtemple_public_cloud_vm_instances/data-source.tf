# List all running VM instances of the tenant (the result set is paginated
# automatically). Filters are optional.
data "cloudtemple_public_cloud_vm_instances" "running" {
  status = "running"
}

output "running_vm_names" {
  value = [for vm in data.cloudtemple_public_cloud_vm_instances.running.instances : vm.name]
}
