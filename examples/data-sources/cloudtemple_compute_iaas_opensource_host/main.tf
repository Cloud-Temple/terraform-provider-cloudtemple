terraform {
  required_providers {
    cloudtemple = {
      source  = "Cloud-Temple/cloudtemple"
      version = "0.15.0"
    }
  }
}

provider "cloudtemple" {
  client_id = ""
  secret_id = ""
}