SHELL := /usr/bin/env bash

.PHONY: fmt gofmt-check terraform-fmt terraform-fmt-check vet test build lint tflint openapi-generate testacc release-artifacts sync-version check-version-drift

fmt:
	go fmt ./...
	terraform fmt -recursive examples

gofmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*' -not -path './vendor/*'))"

terraform-fmt:
	terraform fmt -recursive examples

terraform-fmt-check:
	terraform fmt -check -recursive examples

vet:
	go vet ./...

test:
	go test ./...

build:
	go build ./...

lint:
	golangci-lint run ./...

tflint:
	tflint --chdir=examples/basic-site

openapi-generate:
	./scripts/generate-openapi.sh

sync-version:
	./scripts/sync-version.sh

check-version-drift:
	./scripts/sync-version.sh --check

testacc:
	./scripts/testacc.sh

release-artifacts:
	@test -n "$(VERSION)" || (echo "VERSION is required, for example: make release-artifacts VERSION=0.1.0" >&2; exit 1)
	./scripts/build-release-artifacts.sh "$(VERSION)"
