variable "iam_role_name" {
  type        = string
  description = "Name of the pre-existing IAM role granted to the PAT. PAT create fails with no role."
}

variable "object_storage_role_name" {
  type        = string
  description = "Name of the pre-existing object-storage role granted by the ACL entry (referenced by name)."
}

variable "pat_expiration_date" {
  type        = string
  default     = "2999-01-01T00:00:00Z"
  description = "RFC3339 expiration date for the PAT. Defaults far in the future for a short-lived recette run."
}
