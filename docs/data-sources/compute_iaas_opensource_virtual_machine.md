---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_iaas_opensource_virtual_machine Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a specific virtual machine from an Open IaaS infrastructure.
  To query this datasource you will need the compute_iaas_opensource_read role.
---

# cloudtemple_compute_iaas_opensource_virtual_machine (Data Source)

Used to retrieve a specific virtual machine from an Open IaaS infrastructure.

To query this datasource you will need the `compute_iaas_opensource_read` role.



<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `machine_manager_id` (String)
- `name` (String)

### Read-Only

- `addresses` (List of Object) (see [below for nested schema](#nestedatt--addresses))
- `auto_power_on` (Boolean)
- `boot_order` (List of String)
- `cpu` (Number)
- `dvd_drive` (List of Object) (see [below for nested schema](#nestedatt--dvd_drive))
- `host` (List of Object) (see [below for nested schema](#nestedatt--host))
- `id` (String) The ID of this resource.
- `internal_id` (String)
- `memory` (Number)
- `num_cores_per_socket` (Number)
- `operating_system_name` (String)
- `pool` (List of Object) (see [below for nested schema](#nestedatt--pool))
- `power_state` (String)
- `secure_boot` (Boolean)
- `tools` (List of Object) (see [below for nested schema](#nestedatt--tools))

<a id="nestedatt--addresses"></a>
### Nested Schema for `addresses`

Read-Only:

- `ipv4` (String)
- `ipv6` (String)


<a id="nestedatt--dvd_drive"></a>
### Nested Schema for `dvd_drive`

Read-Only:

- `attached` (Boolean)
- `name` (String)


<a id="nestedatt--host"></a>
### Nested Schema for `host`

Read-Only:

- `id` (String)
- `name` (String)


<a id="nestedatt--pool"></a>
### Nested Schema for `pool`

Read-Only:

- `id` (String)
- `name` (String)


<a id="nestedatt--tools"></a>
### Nested Schema for `tools`

Read-Only:

- `detected` (Boolean)
- `version` (String)

