SHELL := /usr/bin/env bash

DOCS_TRACKED_PATHS := docs templates examples/README.md examples/provider examples/resources examples/data-sources

.PHONY: fmt check actionlint shellcheck govulncheck go-mod-tidy-check gofmt-check terraform-fmt terraform-fmt-check vet test build lint tflint openapi-generate openapi-generated-check openapi-check-upstream docs-generate docs-check testacc release-artifacts sign-release-artifacts sync-version check-version-drift

fmt:
	go fmt ./...
	terraform fmt -recursive examples

check: actionlint shellcheck go-mod-tidy-check govulncheck gofmt-check vet test build openapi-generated-check docs-check terraform-fmt-check check-version-drift

actionlint:
	go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.12

shellcheck:
	shellcheck scripts/*.sh

govulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

go-mod-tidy-check:
	go mod tidy
	git diff --exit-code -- go.mod go.sum

gofmt-check:
	@test -z "$$(gofmt -l $$(find . -name '*.go' -not -path './.git/*' -not -path './.cache/*' -not -path './vendor/*'))"

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

openapi-generated-check:
	./scripts/generate-openapi.sh
	git diff --exit-code -- internal/openapi/generated

openapi-check-upstream:
	./scripts/check-openapi-upstream.sh

docs-generate:
	./scripts/generate-docs.sh

docs-check:
	./scripts/generate-docs.sh
	git diff --exit-code -- $(DOCS_TRACKED_PATHS)

sync-version:
	./scripts/sync-version.sh

check-version-drift:
	./scripts/sync-version.sh --check

testacc:
	./scripts/testacc.sh

release-artifacts:
	@test -n "$(VERSION)" || (echo "VERSION is required, for example: make release-artifacts VERSION=0.1.0" >&2; exit 1)
	./scripts/build-release-artifacts.sh "$(VERSION)"

sign-release-artifacts:
	@test -n "$(VERSION)" || (echo "VERSION is required, for example: make sign-release-artifacts VERSION=0.1.0" >&2; exit 1)
	./scripts/sign-release-artifacts.sh "$(VERSION)"
