# A network adapter is imported by the composite id
# "<virtual_machine_id>/<network_adapter_id>" (both UUIDs). The write-only
# ip_address is not read back and is not part of the imported state.
terraform import cloudtemple_public_cloud_vm_network_adapter.eth1 00000000-0000-0000-0000-000000000000/11111111-1111-1111-1111-111111111111
