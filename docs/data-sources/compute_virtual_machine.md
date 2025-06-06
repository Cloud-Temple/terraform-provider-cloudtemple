---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_virtual_machine Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a specific virtual machine from a vCenter infrastructure.
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_virtual_machine (Data Source)

Used to retrieve a specific virtual machine from a vCenter infrastructure.

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_virtual_machine" "id" {
  id = "de2b8b80-8b90-414a-bc33-e12f61a4c05c"
}

data "cloudtemple_compute_virtual_machine" "name" {
  name = "virtual_machine_67_bob-clone"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `id` (String) The ID of the virtual machine to retrieve. Conflicts with `name`.
- `machine_manager_id` (String) The ID of the machine manager (vCenter) where the virtual machine is located. Required when using `name`.
- `name` (String) The name of the virtual machine to retrieve. Conflicts with `id`. Requires `machine_manager_id`.

### Read-Only

- `boot_options` (List of Object) Boot configuration options for the virtual machine. (see [below for nested schema](#nestedatt--boot_options))
- `consolidation_needed` (Boolean) Whether the virtual machine needs consolidation.
- `cpu` (Number) The number of virtual CPUs allocated to the virtual machine.
- `cpu_hot_add_enabled` (Boolean) Whether CPU hot add is enabled for the virtual machine.
- `cpu_hot_remove_enabled` (Boolean) Whether CPU hot remove is enabled for the virtual machine.
- `cpu_usage` (Number) The current CPU usage of the virtual machine in MHz.
- `datacenter_id` (String) The ID of the datacenter where the virtual machine is located.
- `datastore_cluster_id` (String) The ID of the datastore cluster where the virtual machine is stored.
- `datastore_id` (String) The ID of the datastore where the virtual machine is stored.
- `datastore_name` (String) The name of the datastore where the virtual machine is stored.
- `distributed_virtual_port_group_ids` (List of String) List of distributed virtual port group IDs associated with the virtual machine.
- `expose_hardware_virtualization` (Boolean) Whether hardware virtualization is exposed to the guest operating system.
- `extra_config` (List of Object) Extra configuration parameters for the virtual machine. (see [below for nested schema](#nestedatt--extra_config))
- `guest_operating_system_moref` (String) The managed object reference ID of the guest operating system in the hypervisor.
- `hardware_version` (String) The hardware version of the virtual machine.
- `host_cluster_id` (String) The ID of the host cluster where the virtual machine is running.
- `machine_manager_name` (String) The name of the machine manager (vCenter) where the virtual machine is located.
- `memory` (Number) The amount of memory allocated to the virtual machine in Bytes.
- `memory_hot_add_enabled` (Boolean) Whether memory hot add is enabled for the virtual machine.
- `memory_usage` (Number) The current memory usage of the virtual machine in Bytes.
- `moref` (String) The managed object reference ID of the virtual machine in the hypervisor.
- `num_cores_per_socket` (Number) The number of cores per socket in the virtual machine.
- `operating_system_name` (String) The name of the operating system running on the virtual machine.
- `power_state` (String) The power state of the virtual machine (e.g., poweredOn, poweredOff, suspended).
- `replication_config` (List of Object) Configuration for virtual machine replication. (see [below for nested schema](#nestedatt--replication_config))
- `snapshoted` (Boolean) Whether the virtual machine has snapshots.
- `spp_mode` (String) The SPP (Storage Policy Protection) mode of the virtual machine.
- `storage` (List of Object) Storage usage information for the virtual machine. (see [below for nested schema](#nestedatt--storage))
- `template` (Boolean) Whether the virtual machine is a template.
- `tools` (String) The status of VMware Tools in the virtual machine.
- `tools_version` (Number) The version of VMware Tools installed in the virtual machine.
- `triggered_alarms` (List of Object) List of alarms that have been triggered for this virtual machine. (see [below for nested schema](#nestedatt--triggered_alarms))

<a id="nestedatt--boot_options"></a>
### Nested Schema for `boot_options`

Read-Only:

- `boot_delay` (Number)
- `boot_retry_delay` (Number)
- `boot_retry_enabled` (Boolean)
- `efi_secure_boot_enabled` (Boolean)
- `enter_bios_setup` (Boolean)
- `firmware` (String)


<a id="nestedatt--extra_config"></a>
### Nested Schema for `extra_config`

Read-Only:

- `key` (String)
- `value` (String)


<a id="nestedatt--replication_config"></a>
### Nested Schema for `replication_config`

Read-Only:

- `disk` (List of Object) (see [below for nested schema](#nestedobjatt--replication_config--disk))
- `encryption_destination` (Boolean)
- `generation` (Number)
- `net_compression_enabled` (Boolean)
- `net_encryption_enabled` (Boolean)
- `opp_updates_enabled` (Boolean)
- `paused` (Boolean)
- `quiesce_guest_enabled` (Boolean)
- `rpo` (Number)
- `vm_replication_id` (String)

<a id="nestedobjatt--replication_config--disk"></a>
### Nested Schema for `replication_config.disk`

Read-Only:

- `disk_replication_id` (String)
- `key` (Number)



<a id="nestedatt--storage"></a>
### Nested Schema for `storage`

Read-Only:

- `committed` (Number)
- `uncommitted` (Number)


<a id="nestedatt--triggered_alarms"></a>
### Nested Schema for `triggered_alarms`

Read-Only:

- `id` (String)
- `status` (String)


