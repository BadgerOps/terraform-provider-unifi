SHELL := /usr/bin/env bash

.PHONY: fmt gofmt-check terraform-fmt terraform-fmt-check vet test build lint tflint openapi-generate

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
