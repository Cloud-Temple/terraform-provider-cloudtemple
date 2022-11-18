default: testacc

# Run tests
.PHONY: test
test: testclient testacc  ## Run the tests of the Go client and the Terraform provider

.PHONY: testclient
testclient:  ## Run the integration tests of the Go client
	go test ./internal/client/... -v $(TESTARGS) -timeout 120m

.PHONY: testprovider
testprovider:  ## Run the unit tests of the Terraform provider
	go test ./internal/provider/... -v $(TESTARGS) -timeout 120m

# Run acceptance tests
.PHONY: testacc
testacc:  ## Run the integration tests of the Terraform provider
	TF_ACC=1 go test ./internal/provider/... -v $(TESTARGS) -timeout 120m

.PHONY: testextratime
testextratime:  ## Run the tests that take a long time to complete
	CLIENT_RUN_LONG_TESTS=1 go test ./internal/client/... -v $(TESTARGS) -timeout 120m

.PHONY: fmt
fmt:  ## Run all Go and Terraform files
	go fmt ./...
	terraform fmt -recursive

.PHONY: build
build:  ## Build the Terraform provider
	go build

.PHONY: generate
generate:  ## Generate the documentation
	go generate

.PHONY: help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
