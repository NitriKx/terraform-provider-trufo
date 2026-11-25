default: build

.PHONY: build test testacc generate install fmt

build: ## Build the provider binary
	go build -o terraform-provider-trufo .

test: ## Unit tests (both modules)
	go test ./...
	cd trufo-go && go test ./...

testacc: ## Acceptance tests against the Trufo test host (needs TF_ACC=1 + TRUFO_* env)
	TF_ACC=1 go test ./internal/provider/ -run TestAcc -v -timeout 20m

generate: ## Regenerate docs/ and format examples
	go run github.com/hashicorp/terraform-plugin-docs/cmd/tfplugindocs generate --provider-name trufo
	terraform fmt -recursive examples/

fmt: ## gofmt everything
	gofmt -w .
	cd trufo-go && gofmt -w .

install: ## go install the provider
	go install .
