# Retrieve the tenant's quota (limits and current usage).
data "cloudtemple_public_cloud_vm_quota" "current" {}

output "vcpu_usage" {
  value = "${data.cloudtemple_public_cloud_vm_quota.current.vcpu_used}/${data.cloudtemple_public_cloud_vm_quota.current.vcpu_limit}"
}

output "storage_headroom_gb" {
  value = data.cloudtemple_public_cloud_vm_quota.current.storage_limit_gb - data.cloudtemple_public_cloud_vm_quota.current.storage_used_gb
}
