---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_compute_iaas_opensource_virtual_disk Resource - terraform-provider-cloudtemple"
subcategory: "Compute"
description: |-
  To manage this resource you will need the following roles:
    - compute_iaas_opensource_management
    - compute_iaas_opensource_read
    - activity_read
---

# cloudtemple_compute_iaas_opensource_virtual_disk (Resource)

To manage this resource you will need the following roles:
  - `compute_iaas_opensource_management`
  - `compute_iaas_opensource_read`
  - `activity_read`

## Example Usage

```terraform
# Exemple with a bootable Read only disk
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-01" {
  name     = "openiaas-disk-01"
  size     = 2 * 1024 * 1024 * 1024
  mode     = "RO"
  bootable = true

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
}

# Exemple with a non-bootable Read/Write disk
resource "cloudtemple_compute_iaas_opensource_virtual_disk" "openiaas-disk-02" {
  name     = "openiaas-disk-02"
  size     = 4 * 1024 * 1024 * 1024
  mode     = "RW"
  bootable = false

  storage_repository_id = data.cloudtemple_compute_iaas_opensource_storage_repository.sr001-clu001-t0001-az05-r-flh1-data13.id
  virtual_machine_id    = cloudtemple_compute_iaas_opensource_virtual_machine.OPENIAAS-TERRAFORM-01.id
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `mode` (String) The mode of the virtual disk. Available values are RW (Read/Write) and RO (Read-Only).
- `name` (String) The name of the virtual disk.
- `size` (Number) The size of the virtual disk in bytes.
- `storage_repository_id` (String) The ID of the storage repository where the virtual disk is stored.
- `virtual_machine_id` (String) The ID of the virtual machine to which the virtual disk is attached.

### Optional

- `bootable` (Boolean) Whether the virtual disk is bootable.

### Read-Only

- `id` (String) The ID of the virtual disk.
- `internal_id` (String) The internal ID of the virtual disk.
- `is_snapshot` (Boolean) Whether the virtual disk is a snapshot.
- `templates` (List of Object) The templates to which the virtual disk is attached. (see [below for nested schema](#nestedatt--templates))
- `usage` (Number) The usage of the virtual disk.
- `virtual_machines` (List of Object) The virtual machines to which the virtual disk is attached. (see [below for nested schema](#nestedatt--virtual_machines))

<a id="nestedatt--templates"></a>
### Nested Schema for `templates`

Read-Only:

- `id` (String)
- `name` (String)
- `read_only` (Boolean)


<a id="nestedatt--virtual_machines"></a>
### Nested Schema for `virtual_machines`

Read-Only:

- `id` (String)
- `name` (String)
- `read_only` (Boolean)

## Import

Import is supported using the following syntax:

```shell
#!/bin/bash

# Import a virtual disk using its ID
terraform import cloudtemple_compute_iaas_opensource_virtual_disk.example 12345678-1234-1234-1234-123456789abc
```
