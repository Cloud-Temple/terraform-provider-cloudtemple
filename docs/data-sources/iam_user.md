---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "cloudtemple_iam_user Data Source - terraform-provider-cloudtemple"
subcategory: "IAM"
description: |-
  Used to retrieve information about a specific user.
  To query this datasource you will need the iam_read role.
---

# cloudtemple_iam_user (Data Source)

Used to retrieve information about a specific user.

To query this datasource you will need the `iam_read` role.

## Example Usage

```terraform
data "cloudtemple_iam_user" "id" {
  id = "37105598-4889-43da-82ea-cf60f2a36aee"
}

data "cloudtemple_iam_user" "name" {
  name = "Rémi Lapeyre"
}

data "cloudtemple_iam_user" "internal_id" {
  internal_id = "7b8ba092-52e3-4c21-a2f5-adca40a80d34"
}

data "cloudtemple_iam_user" "email" {
  email = "remi.lapeyre@lenstra.fr"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Optional

- `email` (String) The email of the user.
- `id` (String) The ID of the user.
- `internal_id` (String) The internal ID of the user.
- `name` (String) The name of the user.

### Read-Only

- `email_verified` (Boolean) Whether the user's email is verified.
- `source` (List of String) The source of the user.
- `source_id` (String) The source ID of the user.
- `type` (String) The type of the user.


