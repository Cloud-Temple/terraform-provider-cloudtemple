---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_backup_jobs Data Source - terraform-provider-cloudtemple"
subcategory: "Backup"
description: |-
  Provides information about backup jobs.
  To query this datasource you will need the backup_iaas_spp_read role.
---

# cloudtemple_backup_jobs (Data Source)

Provides information about backup jobs.

To query this datasource you will need the `backup_iaas_spp_read` role.

## Example Usage

```terraform
data "cloudtemple_backup_jobs" "foo" {}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `id` (String) The ID of this resource.
- `jobs` (List of Object) List of backup jobs. (see [below for nested schema](#nestedatt--jobs))

<a id="nestedatt--jobs"></a>
### Nested Schema for `jobs`

Read-Only:

- `display_name` (String)
- `id` (String)
- `name` (String)
- `policy_id` (String)
- `status` (String)
- `type` (String)


