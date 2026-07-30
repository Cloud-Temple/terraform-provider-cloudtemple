# Retrieve all storage types available to the tenant.
data "cloudtemple_public_cloud_vm_storage_types" "all" {}

output "available_storage_types" {
  value = [
    for st in data.cloudtemple_public_cloud_vm_storage_types.all.storage_types :
    "${st.name} (${st.iops_hint})" if st.is_available
  ]
}

# Narrow the catalogue to an availability-zone/instance-family pair. The two
# filters are optional but must always be set together; picking the family from
# the zone's compatible_families guarantees a valid pair.
data "cloudtemple_public_cloud_vm_availability_zones" "all" {}

data "cloudtemple_public_cloud_vm_storage_types" "for_az_and_family" {
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].id
  instance_family_id   = data.cloudtemple_public_cloud_vm_availability_zones.all.availability_zones[0].compatible_families[0].id
}
