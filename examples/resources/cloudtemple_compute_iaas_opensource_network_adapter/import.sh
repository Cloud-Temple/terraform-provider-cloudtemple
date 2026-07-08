#!/bin/bash

# Import a network adapter using its ID
terraform import cloudtemple_compute_iaas_opensource_network_adapter.example 12345678-1234-1234-1234-123456789abc

# Connecting a VM's adapter to a VPC at provisioning time:
# A VM created from a marketplace item or a template gets its network adapter(s)
# declared inline on the cloudtemple_compute_iaas_opensource_virtual_machine
# resource, which cannot assign a VPC static IP. To give such an adapter a VPC
# static IP, hand it off to this standalone resource: set
# `lifecycle { ignore_changes = [os_network_adapter] }` on the VM (so it stops
# managing the adapter without deleting it), then import the adapter here with
# `network_id` pointing at a VPC-backed network and `ip_address` set. The next
# apply moves the existing adapter onto the VPC in place (no re-creation).
#
#   terraform import cloudtemple_compute_iaas_opensource_network_adapter.adopted <adapter-id>
