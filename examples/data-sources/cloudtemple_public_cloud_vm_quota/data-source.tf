# Retrieve the tenant's Public Cloud VM Instances quota (limits + usage).
data "cloudtemple_public_cloud_vm_quota" "current" {}

output "vcpu_free" {
  value = data.cloudtemple_public_cloud_vm_quota.current.vcpu_limit - data.cloudtemple_public_cloud_vm_quota.current.vcpu_used
}
