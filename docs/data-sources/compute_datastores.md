---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_datastores Data Source - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  Used to retrieve a list of datastores.
  To query this datasource you will need the compute_iaas_vmware_read role.
---

# cloudtemple_compute_datastores (Data Source)

Used to retrieve a list of datastores.

To query this datasource you will need the `compute_iaas_vmware_read` role.

## Example Usage

```terraform
data "cloudtemple_compute_datastores" "foo" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `datacenter_id` (String) Filter datastores by datacenter ID.
- `datastore_cluster_id` (String) Filter datastores by datastore cluster ID.
- `host_cluster_id` (String) Filter datastores by host cluster ID.
- `host_id` (String) Filter datastores by host ID.
- `machine_manager_id` (String) Filter datastores by machine manager ID.
- `name` (String) Filter datastores by name.

### Read-Only

- `datastores` (List of Object) List of datastores matching the filter criteria. (see [below for nested schema](#nestedatt--datastores))
- `id` (String) The ID of this resource.

<a id="nestedatt--datastores"></a>
### Nested Schema for `datastores`

Read-Only:

- `accessible` (Number)
- `associated_folder` (String)
- `free_capacity` (Number)
- `hosts_names` (List of String)
- `hosts_number` (Number)
- `id` (String)
- `machine_manager_id` (String)
- `maintenance_status` (Boolean)
- `max_capacity` (Number)
- `moref` (String)
- `name` (String)
- `type` (String)
- `unique_id` (String)
- `virtual_machines_number` (Number)


