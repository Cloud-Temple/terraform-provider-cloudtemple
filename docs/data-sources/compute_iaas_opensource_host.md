---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_iaas_opensource_host Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a specific host from an Open IaaS infrastructure.
  To query this datasource you will need the compute_iaas_opensource_infrastructure_read role.
---

# cloudtemple_compute_iaas_opensource_host (Data Source)

Used to retrieve a specific host from an Open IaaS infrastructure.

To query this datasource you will need the `compute_iaas_opensource_infrastructure_read` role.

## Example Usage

```terraform
# This example demonstrates how to use the data source to get the host details.
# The data source is used to fetch the host details based on the host ID or name and availability zone.

# Exemple with the ID of the host.
data "cloudtemple_compute_iaas_opensource_host" "host-1" {
  id = "host_id"
}

output "host-1" {
  value = data.cloudtemple_compute_iaas_opensource_host.host
}

# Exemple with the name of the host.
# See the availability zone data source in the examples/data-sources/cloudtemple_compute_iaas_opensource_availability_zone/data-source.tf file.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_compute_iaas_opensource_host" "host-2" {
  name               = "host_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "host-2" {
  value = data.cloudtemple_compute_iaas_opensource_host.host-2
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `id` (String) The ID of the host to retrieve. Conflicts with `name`.
- `machine_manager_id` (String) The ID of the machine manager this host belongs to.
- `name` (String) The name of the host to retrieve. Conflicts with `id`.
- `pool_id` (String) The ID of the pool this host belongs to.

### Read-Only

- `internal_id` (String)
- `master` (Boolean)
- `metrics` (List of Object) (see [below for nested schema](#nestedatt--metrics))
- `pool` (List of Object) (see [below for nested schema](#nestedatt--pool))
- `power_state` (String)
- `reboot_required` (Boolean)
- `update_data` (List of Object) (see [below for nested schema](#nestedatt--update_data))
- `uptime` (Number)
- `virtual_machines` (List of String)

<a id="nestedatt--metrics"></a>
### Nested Schema for `metrics`

Read-Only:

- `cpu` (List of Object) (see [below for nested schema](#nestedobjatt--metrics--cpu))
- `memory` (List of Object) (see [below for nested schema](#nestedobjatt--metrics--memory))
- `xoa` (List of Object) (see [below for nested schema](#nestedobjatt--metrics--xoa))

<a id="nestedobjatt--metrics--cpu"></a>
### Nested Schema for `metrics.cpu`

Read-Only:

- `cores` (Number)
- `model` (String)
- `model_name` (String)
- `sockets` (Number)


<a id="nestedobjatt--metrics--memory"></a>
### Nested Schema for `metrics.memory`

Read-Only:

- `size` (Number)
- `usage` (Number)


<a id="nestedobjatt--metrics--xoa"></a>
### Nested Schema for `metrics.xoa`

Read-Only:

- `build` (String)
- `full_name` (String)
- `version` (String)



<a id="nestedatt--pool"></a>
### Nested Schema for `pool`

Read-Only:

- `id` (String)
- `name` (String)
- `type` (List of Object) (see [below for nested schema](#nestedobjatt--pool--type))

<a id="nestedobjatt--pool--type"></a>
### Nested Schema for `pool.type`

Read-Only:

- `description` (String)
- `key` (String)



<a id="nestedatt--update_data"></a>
### Nested Schema for `update_data`

Read-Only:

- `maintenance_mode` (Boolean)
- `status` (String)


