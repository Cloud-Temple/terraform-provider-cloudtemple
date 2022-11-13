# Read a virtual switch using its ID
data "cloudtemple_compute_virtual_switch" "id" {
  id = "6e7b457c-bdb1-4272-8abf-5fd6e9adb8a4"
}

# Read a virtual switch using its name
data "cloudtemple_compute_virtual_switch" "name" {
  name = "dvs002-ucs01_FLO-DC-EQX6"
}
