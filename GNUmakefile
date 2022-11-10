default: testacc

# Run tests
.PHONY: test
test: testclient testacc

.PHONY: testclient
testclient:
	go test ./internal/client/... -v $(TESTARGS) -timeout 120m

# Run acceptance tests
.PHONY: testacc
testacc:
	TF_ACC=1 go test ./internal/provider/... -v $(TESTARGS) -timeout 120m

.PHONY: fmt
fmt:
	go fmt ./...
	terraform fmt -recursive

.PHONY: build
build:
	go build

.PHONY: generate
generate:
	go generate