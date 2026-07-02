# A VM instance is imported by its UUID. cloud_init and os_network_adapter are
# not returned by the API and cannot be reconciled on import; power_state is
# derived from the VM status.
terraform import cloudtemple_public_cloud_vm_instance.web 00000000-0000-0000-0000-000000000000
