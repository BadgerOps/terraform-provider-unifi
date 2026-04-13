# PR: Align docs to plan and enforce version drift checks

## Summary

This PR removes the temporary migration-focused documentation, realigns the repository docs with the shared BadgerOps plan and the committed OpenAPI snapshot, and adds pre-commit plus CI checks to prevent version drift across checked-in examples and local validation wiring.

## Why

Before this PR:

- the repo still carried a migration document that was only intended as a temporary test artifact
- the README, Terraform example, and Terraform validation workflow had drifted to different provider versions
- there was no repo-managed pre-commit or CI guardrail to keep those checked-in version references aligned with `CHANGELOG.md`

This PR addresses those gaps by:

- removing the migration document and references to it
- treating the shared plan plus the committed OpenAPI snapshot as the documented source of truth
- deriving versioned examples and local validation wiring from the current changelog release

## Main Changes

### 1. Documentation aligned to the plan and OpenAPI snapshot

The README now points at the shared BadgerOps plan and the committed UniFi OpenAPI snapshot instead of the removed migration guidance.

Changes include:

- removing the migration section and `docs/MIGRATION.md`
- updating README wording to match the current project direction
- updating checked-in version examples to the current release

Relevant files:

- [README.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/README.md)
- [docs/MIGRATION.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/MIGRATION.md)

### 2. Version drift checks added for pre-commit and CI

This PR adds a repo-local version sync/check script that derives the current provider version from `CHANGELOG.md` and keeps versioned references aligned.

The new check covers:

- README provider example version
- README local build example version
- README `make release-artifacts` example
- `examples/basic-site/main.tf`

It is wired into:

- `make sync-version`
- `make check-version-drift`
- local `pre-commit`
- GitHub Actions `go.yml`

Relevant files:

- [scripts/sync-version.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/scripts/sync-version.sh)
- [.pre-commit-config.yaml](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.pre-commit-config.yaml)
- [Makefile](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/Makefile)
- [.github/workflows/go.yml](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.github/workflows/go.yml)
- [flake.nix](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/flake.nix)

### 3. Terraform example validation now follows the changelog version

The Terraform validation workflow no longer hardcodes the provider mirror version. It reads the current release version from `CHANGELOG.md` before building the local mirror used by `terraform init` and `terraform validate`.

Relevant files:

- [.github/workflows/terraform.yml](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.github/workflows/terraform.yml)
- [scripts/read-changelog-release.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/scripts/read-changelog-release.sh)

### 4. Shared plan updated with follow-up TODO

The shared plan now carries an explicit short-term TODO for repo-local pre-commit and CI checks that keep versioned references synchronized from `CHANGELOG.md`.

Relevant file:

- [PLAN.md](/home/badger/code/badgerops-unifi-provider/PLAN.md)

## Validation

Validated locally:

- `make check-version-drift`
- `go test ./...`
- `terraform fmt -check -recursive examples`
