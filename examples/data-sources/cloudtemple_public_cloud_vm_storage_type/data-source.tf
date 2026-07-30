# Retrieve a storage type by name.
data "cloudtemple_public_cloud_vm_storage_type" "fast" {
  name = "Enterprise"
}

# The storage type id is consumed by the data-disk resource.
output "storage_type" {
  value = {
    id          = data.cloudtemple_public_cloud_vm_storage_type.fast.id
    iops_hint   = data.cloudtemple_public_cloud_vm_storage_type.fast.iops_hint
    max_size_gb = data.cloudtemple_public_cloud_vm_storage_type.fast.max_size_gb
  }
}

# Resolve the name within an availability-zone/instance-family pair. The two
# filters are optional but must always be set together; picking the family from
# the zone's compatible_families guarantees a valid pair.
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

data "cloudtemple_public_cloud_vm_storage_type" "fast_in_az" {
  name                 = "Enterprise"
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].id
  instance_family_id   = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].compatible_families[0].id
}
