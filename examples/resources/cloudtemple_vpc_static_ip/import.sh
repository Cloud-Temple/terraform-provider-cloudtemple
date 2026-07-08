#!/bin/bash

# Import a custom VPC static IP using its ID.
# Only "custom" static IPs (allocated by this resource) can be imported;
# importing a platform-managed static IP (e.g. source "xoa") is rejected.
terraform import cloudtemple_vpc_static_ip.example 12345678-1234-1234-1234-123456789abc
