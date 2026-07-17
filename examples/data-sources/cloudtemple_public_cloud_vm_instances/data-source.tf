# List all VM instances of the tenant (paginated automatically).
data "cloudtemple_public_cloud_vm_instances" "all" {}

# Server-side filters: name, status, availability_zone_id, instance_family_id.
data "cloudtemple_public_cloud_vm_instances" "running" {
  status = "running"
}

output "running_vms" {
  value = [for vm in data.cloudtemple_public_cloud_vm_instances.running.instances : vm.name]
}
