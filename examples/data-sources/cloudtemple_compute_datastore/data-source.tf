# Read a datastore using its ID
data "cloudtemple_compute_datastore" "id" {
  id = "d439d467-943a-49f5-a022-c0c25b737022"
}

# Read a datastore using its name
data "cloudtemple_compute_datastore" "name" {
  name = "ds001-bob-svc1-data4-eqx6"
}