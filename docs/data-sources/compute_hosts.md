---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_hosts Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a list of hosts.
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_hosts (Data Source)

Used to retrieve a list of hosts.

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_hosts" "foo" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `datacenter_id` (String) Filter hosts by datacenter ID.
- `datastore_id` (String) Filter hosts by datastore ID.
- `host_cluster_id` (String) Filter hosts by host cluster ID.
- `machine_manager_id` (String) Filter hosts by machine manager ID.
- `name` (String) Filter hosts by name.

### Read-Only

- `hosts` (List of Object) List of hosts matching the filter criteria. (see [below for nested schema](#nestedatt--hosts))
- `id` (String) The ID of this resource.

<a id="nestedatt--hosts"></a>
### Nested Schema for `hosts`

Read-Only:

- `id` (String)
- `machine_manager_id` (String)
- `metrics` (List of Object) (see [below for nested schema](#nestedobjatt--hosts--metrics))
- `moref` (String)
- `name` (String)
- `virtual_machines` (List of Object) (see [below for nested schema](#nestedobjatt--hosts--virtual_machines))

<a id="nestedobjatt--hosts--metrics"></a>
### Nested Schema for `hosts.metrics`

Read-Only:

- `connected` (Boolean)
- `cpu` (List of Object) (see [below for nested schema](#nestedobjatt--hosts--metrics--cpu))
- `esx` (List of Object) (see [below for nested schema](#nestedobjatt--hosts--metrics--esx))
- `maintenance_status` (Boolean)
- `memory` (List of Object) (see [below for nested schema](#nestedobjatt--hosts--metrics--memory))
- `uptime` (Number)

<a id="nestedobjatt--hosts--metrics--cpu"></a>
### Nested Schema for `hosts.metrics.cpu`

Read-Only:

- `cpu_cores` (Number)
- `cpu_mhz` (Number)
- `cpu_threads` (Number)
- `overall_cpu_usage` (Number)


<a id="nestedobjatt--hosts--metrics--esx"></a>
### Nested Schema for `hosts.metrics.esx`

Read-Only:

- `build` (Number)
- `full_name` (String)
- `version` (String)


<a id="nestedobjatt--hosts--metrics--memory"></a>
### Nested Schema for `hosts.metrics.memory`

Read-Only:

- `memory_size` (Number)
- `memory_usage` (Number)



<a id="nestedobjatt--hosts--virtual_machines"></a>
### Nested Schema for `hosts.virtual_machines`

Read-Only:

- `id` (String)
- `type` (String)


