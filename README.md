# Terraform Provider Cloud-Temple

## Requirements

-	[Terraform](https://www.terraform.io/downloads.html) >= 0.13.x
-	[Go](https://golang.org/doc/install) >= 1.18


## Using the provider

```terraform
provider "cloudtemple" {
  # Can also be set as the CLOUDTEMPLE_CLIENT_ID environment variable
  client_id = "2f31d624-e4b5-43a5-a41f-10abe0267400"

  # Can also be set as the CLOUDTEMPLE_SECRET_ID environment variable
  secret_id = "45f25b78-ae4f-4146-85e0-6627ab91047d"
}
```

- `address` (String) The HTTP address to connect to the API. Defaults to `shiva.cloud-temple.com`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_ADDR`.
- `client_id` (String) The client ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_CLIENT_ID`.
- `scheme` (String) The URL scheme to used to connect to the API. Default to `https`. Can also be specified with the environment variable `CLOUDTEMPLE_HTTP_SCHEME`.
- `secret_id` (String, Sensitive) The secret ID to login to the API with. Can also be specified with the environment variable `CLOUDTEMPLE_SECRET_ID`.


## Developing the Provider

If you wish to work on the provider, you'll first need [Go](http://www.golang.org) installed on your machine (see [Requirements](#requirements) above).

To compile the provider, run `go install`. This will build the provider and put the provider binary in the `$GOPATH/bin` directory.

To generate or update documentation, run `go generate`.

## Acceptance tests

In order to run the tests suite, rename .env.test.dist to .env.test in root directory. Fill the ENV vars with existing correct data from the Shiva instance you whant to work. Use the shiva interface or the api to get the data or contact a project administrator for information.


To run client tests

```sh
$ make testclient
```

To run provider tests

```sh
$ make testprovider
```

To run full acceptance testes

```sh
$ make testacc
```
See GNUmakefile in root directory to more test cases.

*Note:* Acceptance tests create real resources, and often cost money to run according on the shiva instance where you lunch them.
