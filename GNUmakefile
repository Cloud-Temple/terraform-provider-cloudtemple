default: testacc

# Run tests
.PHONY: test
test:
	go test ./... -v $(TESTARGS) -timeout 120m

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./... -v $(TESTARGS) -timeout 120m

.PHONY: fmt
fmt:
	go fmt ./...
	terraform fmt -recursive

.PHONY: build
build:
	go build