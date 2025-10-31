data "cloudtemple_marketplace_item" "example_by_id" {
  id = "12345678-1234-1234-1234-123456789012"
}

data "cloudtemple_marketplace_item" "example_by_name" {
  name = "Ubuntu 22.04 LTS"
}

output "item_id" {
  value = data.cloudtemple_marketplace_item.example_by_name.id
}

output "item_description" {
  value = data.cloudtemple_marketplace_item.example_by_name.description
}

output "item_version" {
  value = data.cloudtemple_marketplace_item.example_by_name.version
}

output "deployment_targets" {
  value = data.cloudtemple_marketplace_item.example_by_name.deployment_options[0].targets
}
