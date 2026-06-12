# Terraform Provider for Cloud Temple

The official [Terraform](https://www.terraform.io) provider to manage the
resources of your [Cloud Temple](https://www.cloud-temple.com) account.

- Terraform Registry: [Cloud-Temple/cloudtemple](https://registry.terraform.io/providers/Cloud-Temple/cloudtemple/latest)
- Provider documentation: [registry.terraform.io/providers/Cloud-Temple/cloudtemple/latest/docs](https://registry.terraform.io/providers/Cloud-Temple/cloudtemple/latest/docs)
- Changelog: [CHANGELOG.md](CHANGELOG.md)

## Requirements

- [Terraform](https://developer.hashicorp.com/terraform/downloads) >= 0.13.x
- [Go](https://golang.org/doc/install) >= 1.22 (to build the provider)

## Using the provider

```terraform
terraform {
  required_providers {
    cloudtemple = {
      source  = "Cloud-Temple/cloudtemple"
      version = "~> 1.7"
    }
  }
}

provider "cloudtemple" {
  # Can also be set with the CLOUDTEMPLE_CLIENT_ID environment variable
  client_id = "12345678-1234-1234-1234-123456789abc"

  # Can also be set with the CLOUDTEMPLE_SECRET_ID environment variable
  secret_id = "12345678-1234-1234-1234-123456789abc"
}
```

### Provider arguments

- `client_id` (String, Required) The client ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_CLIENT_ID`.
- `secret_id` (String, Sensitive, Required) The secret ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_SECRET_ID`.
- `address` (String, Optional) The HTTP address to connect to the API. Defaults to `shiva.cloud-temple.com`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_ADDR`.
- `scheme` (String, Optional) The URL scheme used to connect to the API. Defaults to `https`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_SCHEME`.
- `api_suffix` (Boolean, Optional) Specify whether it is necessary to use an `/api` suffix after the address. (Used for development purposes only.)

See the [provider documentation](https://registry.terraform.io/providers/Cloud-Temple/cloudtemple/latest/docs)
for the full list of resources and data sources, and for logging options.

## Developing the provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org)
installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put
the provider binary in the `$GOPATH/bin` directory. You can also use:

```sh
make build
```

To generate or update the documentation, run:

```sh
make generate
```

## Tests

In order to run the test suite, rename `.env.test.dist` to `.env.test` in the
root directory and fill the environment variables with existing, correct data
from the Shiva instance you want to work with. Use the Shiva console or the API
to get the data, or contact a project administrator for more information.

To run the Go client tests:

```sh
make testclient
```

To run the provider unit tests:

```sh
make testprovider
```

To run the full acceptance tests:

```sh
make testacc
```

See the `GNUmakefile` in the root directory for more test targets.

*Note:* acceptance tests create real resources, and often cost money to run
depending on the Shiva instance where you launch them.

## License

This provider is distributed under the [Mozilla Public License 2.0](LICENSE).
