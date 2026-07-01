# Retrieve a single VM instance by its id.
data "cloudtemple_public_cloud_vm_instance" "by_id" {
  id = "00000000-0000-0000-0000-000000000000"
}

output "vm_status" {
  value = data.cloudtemple_public_cloud_vm_instance.by_id.status
}
