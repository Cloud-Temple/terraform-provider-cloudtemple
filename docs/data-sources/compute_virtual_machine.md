---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_virtual_machine Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_virtual_machine (Data Source)

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

- `name` (String)

### Read-Only

- `boot_options` (List of Object) (see [below for nested schema](#nestedatt--boot_options))
- `consolidation_needed` (Boolean)
- `cpu` (Number)
- `cpu_hot_add_enabled` (Boolean)
- `cpu_hot_remove_enabled` (Boolean)
- `cpu_usage` (Number)
- `datacenter_id` (String)
- `datastore_cluster_id` (String)
- `datastore_id` (String)
- `datastore_name` (String)
- `distributed_virtual_port_group_ids` (List of String)
- `expose_hardware_virtualization` (Boolean)
- `extra_config` (List of Object) (see [below for nested schema](#nestedatt--extra_config))
- `guest_operating_system_moref` (String)
- `hardware_version` (String)
- `host_cluster_id` (String)
- `id` (String) The ID of this resource.
- `machine_manager_id` (String)
- `machine_manager_name` (String)
- `machine_manager_type` (String)
- `memory` (Number)
- `memory_hot_add_enabled` (Boolean)
- `memory_usage` (Number)
- `moref` (String)
- `num_cores_per_socket` (Number)
- `operating_system_name` (String)
- `power_state` (String)
- `replication_config` (List of Object) (see [below for nested schema](#nestedatt--replication_config))
- `snapshoted` (Boolean)
- `spp_mode` (String)
- `storage` (List of Object) (see [below for nested schema](#nestedatt--storage))
- `template` (Boolean)
- `tools` (String)
- `tools_version` (Number)
- `triggered_alarms` (List of Object) (see [below for nested schema](#nestedatt--triggered_alarms))

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


