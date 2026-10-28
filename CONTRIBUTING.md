# Contributing to terraform-provider-unifi

Thank you for helping improve the UniFi Terraform provider. Bug reports, focused fixes, documentation improvements, and new provider capabilities are welcome.

## Before you start

- Search the existing issues and pull requests before opening a duplicate.
- Open an issue before starting a large feature or a change to public resource behavior.
- Report suspected vulnerabilities through [GitHub private vulnerability reporting](https://github.com/BadgerOps/terraform-provider-unifi/security/advisories/new), not a public issue.

## Development setup

The preferred environment is the pinned Nix development shell:

```bash
nix develop
```

Without Nix, install the Go version declared in `go.mod`, Terraform, GNU Make, TFLint, and golangci-lint. The commands used by CI are exposed through the `Makefile`.

Create a topic branch in your fork, keep the change focused, and avoid unrelated formatting or generated-file churn.

## Changelog and versions

Add user-visible changes to the appropriate section under `## [Unreleased]` in `CHANGELOG.md`.

Do not create a versioned changelog heading, bump provider versions, or modify release metadata in an ordinary contribution. Maintainers prepare version changes in a dedicated release pull request.

## Tests and checks

Run the checks relevant to your change. The usual local set is:

```bash
make gofmt-check
make check-version-drift
make vet
make test
make build
make lint
make terraform-fmt-check
make tflint
make docs-check
```

Add or update tests for behavior changes. Provider schema changes should include Terraform-level coverage where practical, not only helper-unit tests.

### Generated code and documentation

- Update provider examples when public configuration changes.
- Run `make docs-generate` and commit the resulting documentation.
- Change the OpenAPI snapshot or generated client only through the documented workflow in `internal/openapi/README.md`.
- Do not edit generated files by hand.

### Live acceptance tests

Live tests modify a real UniFi controller and run only with `TF_ACC=1`. Use an isolated test site and credentials that cannot affect production networks. See `.env.example` and run:

```bash
make testacc
```

Live credentials, API keys, controller URLs, state files, and unredacted debug logs must never be committed or pasted into an issue or pull request.

## Pull requests

A pull request should:

- Explain the user-visible problem and the chosen solution.
- Link related issues.
- Include tests and generated documentation when applicable.
- Update `CHANGELOG.md` under `Unreleased` for user-visible changes.
- Call out breaking changes, state migrations, imports, or controller-version assumptions.
- Pass all required GitHub Actions checks.
- Respond to review comments and resolve conversations before merge.

Maintainers may ask for a smaller scope when a pull request combines independent changes. By contributing, you agree that your contribution is provided under this repository's MIT license.
