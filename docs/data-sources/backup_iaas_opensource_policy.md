---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_backup_iaas_opensource_policy Data Source - terraform-provider-cloudtemple"
subcategory: "Backup"
description: |-
  Used to retrieve a specific backup policy from an Open IaaS infrastructure.
  To query this datasource you will need the backup_iaas_opensource_read role.
---

# cloudtemple_backup_iaas_opensource_policy (Data Source)

Used to retrieve a specific backup policy from an Open IaaS infrastructure.

To query this datasource you will need the `backup_iaas_opensource_read` role.

## Example Usage

```terraform
# This example demonstrates how to use the data source to get the policy details.
# The data source is used to fetch the policy details based on the policy ID or name.

# Exemple with the ID of the policy.
data "cloudtemple_backup_iaas_opensource_policy" "policy-1" {
  id = "policy_id"
}

output "policy-1" {
  value = data.cloudtemple_backup_iaas_opensource_policy.policy-1
}

# Exemple with the name of the policy.
data "cloudtemple_compute_iaas_opensource_availability_zone" "availability_zone" {
  name = "availability_zone_name"
}

data "cloudtemple_backup_iaas_opensource_policy" "policy-2" {
  name               = "policy_name"
  machine_manager_id = data.cloudtemple_compute_iaas_opensource_availability_zone.availability_zone.id
}

output "policy-2" {
  value = data.cloudtemple_backup_iaas_opensource_policy.policy-2
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `id` (String) The ID of the backup policy to retrieve. Conflicts with `name`.
- `machine_manager_id` (String) The ID of the machine manager to filter policies by. Required when using `name`.
- `name` (String) The name of the backup policy to retrieve. Conflicts with `id`.

### Read-Only

- `internal_id` (String) The internal identifier of the policy in the Open IaaS system.
- `machine_manager_name` (String) The name of the machine manager associated with this policy.
- `mode` (String) The backup mode of the policy (e.g., full, incremental).
- `running` (Boolean) Indicates whether the policy is currently running.
- `schedulers` (List of Object) List of schedulers configured for this backup policy. (see [below for nested schema](#nestedatt--schedulers))
- `virtual_machines` (List of String) List of virtual machines associated with this backup policy.

<a id="nestedatt--schedulers"></a>
### Nested Schema for `schedulers`

Read-Only:

- `cron` (String)
- `retention` (Number)
- `temporarily_disabled` (Boolean)
- `timezone` (String)


