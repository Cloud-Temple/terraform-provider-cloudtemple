# Read a SPP server using its ID
data "cloudtemple_backup_spp_server" "id" {
  id = "a3d46fb5-29af-4b98-a665-1e82a62fd6d3"
}

# Read a SPP server using its name
data "cloudtemple_backup_spp_server" "name" {
  name = "10"
}