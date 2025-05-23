---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_backup_storages Data Source - terraform-provider-cloudtemple"
subcategory: "Backup"
description: |-
  Used to retrieve a list of backup storage systems.
  To query this datasource you will need the backup_iaas_spp_read role.
---

# cloudtemple_backup_storages (Data Source)

Used to retrieve a list of backup storage systems.

To query this datasource you will need the `backup_iaas_spp_read` role.

## Example Usage

```terraform
data "cloudtemple_backup_storages" "foo" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `id` (String) The ID of this resource.
- `storages` (List of Object) List of backup storage systems. (see [below for nested schema](#nestedatt--storages))

<a id="nestedatt--storages"></a>
### Nested Schema for `storages`

Read-Only:

- `capacity` (List of Object) (see [below for nested schema](#nestedobjatt--storages--capacity))
- `host_address` (String)
- `id` (String)
- `initialize_status` (String)
- `is_ready` (Boolean)
- `name` (String)
- `port_number` (Number)
- `resource_type` (String)
- `site` (String)
- `ssl_connection` (Boolean)
- `storage_id` (String)
- `type` (String)
- `version` (String)

<a id="nestedobjatt--storages--capacity"></a>
### Nested Schema for `storages.capacity`

Read-Only:

- `free` (Number)
- `total` (Number)
- `update_time` (Number)


