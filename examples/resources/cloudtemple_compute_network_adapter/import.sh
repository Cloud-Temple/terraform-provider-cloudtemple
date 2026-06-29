$ terraform import cloudtemple_compute_network_adapter.foo c74060bf-ebb3-455a-b0b0-d0dcb79f3d86

# After importing an existing adapter, you can connect it to a VPC by pointing
# network_id at a VPC-backed network and setting ip_address: the next apply moves
# the adapter onto the VPC in place and assigns the chosen static IP.
