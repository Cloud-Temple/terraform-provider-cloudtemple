# Create a floating IP without binding
resource "cloudtemple_vpc_floating_ip" "example" {
}

# Create a floating IP with a description
resource "cloudtemple_vpc_floating_ip" "with_description" {
  description = "Floating IP for production web server"
}

# Create a floating IP bound to a static IP
resource "cloudtemple_vpc_floating_ip" "with_static_ip" {
  description  = "Floating IP bound to static IP"
  static_ip_id = "xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx"
}

# Output the floating IP details
output "floating_ip_id" {
  description = "The ID of the floating IP"
  value       = cloudtemple_vpc_floating_ip.example.id
}

output "floating_ip_address" {
  description = "The IP address assigned to the floating IP"
  value       = cloudtemple_vpc_floating_ip.example.ip_address
}

output "floating_ip_vpc_id" {
  description = "The VPC ID associated with the floating IP"
  value       = cloudtemple_vpc_floating_ip.example.vpc_id
}

output "bound_static_ip_id" {
  description = "The static IP ID bound to the floating IP"
  value       = cloudtemple_vpc_floating_ip.with_static_ip.static_ip_id
}
