data "cloudtemple_public_cloud_vm_availability_zone" "az" {
  name = "fr1-az01"
}

data "cloudtemple_public_cloud_vm_instance_family" "family" {
  name = "Development"
}

# Template names are not guaranteed unique across zones and families: combine
# the name with the server-side filters to select the right image. An ambiguous
# match fails with the candidate ids instead of silently picking one.
data "cloudtemple_public_cloud_vm_template" "os" {
  name                 = "Debian 13"
  availability_zone_id = data.cloudtemple_public_cloud_vm_availability_zone.az.id
  instance_family_id   = data.cloudtemple_public_cloud_vm_instance_family.family.id
}

output "template" {
  value = {
    id         = data.cloudtemple_public_cloud_vm_template.os.id
    os_family  = data.cloudtemple_public_cloud_vm_template.os.os_family
    os_version = data.cloudtemple_public_cloud_vm_template.os.os_version
  }
}
