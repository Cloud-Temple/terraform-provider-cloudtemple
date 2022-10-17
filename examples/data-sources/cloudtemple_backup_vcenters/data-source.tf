data "cloudtemple_backup_spp_server" "spp_server" {
  name = "10"
}

data "cloudtemple_backup_vcenters" "vcenters" {
  spp_server_id = data.cloudtemple_backup_spp_server.spp_server.id
}