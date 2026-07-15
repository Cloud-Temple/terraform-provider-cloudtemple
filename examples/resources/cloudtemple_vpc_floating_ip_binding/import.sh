#!/bin/bash

# Import a floating-IP <-> static-IP binding using "<floating_ip_id>/<static_ip_id>".
terraform import cloudtemple_vpc_floating_ip_binding.public 12345678-1234-1234-1234-123456789abc/87654321-4321-4321-4321-cba987654321
