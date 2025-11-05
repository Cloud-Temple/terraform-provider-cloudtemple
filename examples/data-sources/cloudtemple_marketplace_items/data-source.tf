data "cloudtemple_marketplace_items" "all" {
}

data "cloudtemple_marketplace_items" "ubuntu" {
  name = "ubuntu"
}

output "all_items_count" {
  value = length(data.cloudtemple_marketplace_items.all.marketplace_items)
}

output "ubuntu_items" {
  value = [for item in data.cloudtemple_marketplace_items.ubuntu.marketplace_items : {
    id      = item.id
    name    = item.name
    version = item.version
    editor  = item.editor
  }]
}
