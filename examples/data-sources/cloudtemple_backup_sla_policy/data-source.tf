# Read an SLA policy using its ID
data "cloudtemple_backup_sla_policy" "id" {
  id = "442718ef-44a1-43d7-9b57-2d910d74e928"
}

# Read an SLA policy using its name
data "cloudtemple_backup_sla_policy" "name" {
  name = "SLA_ADMIN"
}