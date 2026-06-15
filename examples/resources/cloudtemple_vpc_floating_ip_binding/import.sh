#!/bin/bash

# Import a VPC floating IP binding using the composite id "{floating_ip_id}:{static_ip_id}".
terraform import cloudtemple_vpc_floating_ip_binding.example a1b2c3d4-e5f6-7890-1234-567890abcdef:b2c3d4e5-f678-9012-3456-7890abcdef12
