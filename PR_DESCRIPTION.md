# PR: Generate Provider Docs And Enforce Docs Drift Checks

## Summary

This PR adds generated provider documentation for the current UniFi Terraform surface, checks in the example corpus that drives those docs, and wires doc generation into both local contributor workflow and CI.

## Why

Before this PR:

- the repo had no generated provider docs under `docs/`
- there was no checked-in contract for provider, resource, data source, and import examples used for documentation
- there was no dedicated docs generation workflow in CI
- contributors had no local guardrail to catch docs drift before pushing changes

This PR addresses that by:

- generating markdown docs from the provider schema and checked-in examples
- checking in the source inputs used by `tfplugindocs`
- documenting the generation workflow in the repo itself
- enforcing docs drift checks in both `pre-commit` and GitHub Actions

## Main Changes

### 1. Generated provider documentation added under `docs/`

The repo now checks in generated markdown docs for the provider, all supported resources, and all supported data sources.

This includes:

- a generated provider landing page with scope and workflow guidance
- generated resource documentation with example usage and import examples
- generated data source documentation with example usage

Relevant files:

- [docs/index.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/index.md)
- [docs/resources/network.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/resources/network.md)
- [docs/resources/firewall_policy.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/resources/firewall_policy.md)
- [docs/resources/wifi_broadcast.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/resources/wifi_broadcast.md)
- [docs/data-sources/site.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/data-sources/site.md)
- [docs/data-sources/network.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/data-sources/network.md)
- [docs/data-sources/wifi_broadcast.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/data-sources/wifi_broadcast.md)

### 2. Docs generation inputs checked in and documented

The repo now carries the example corpus and template input used to render the generated markdown.

This includes:

- provider example input
- resource examples and import examples
- data source examples
- a custom provider index template
- an examples README that explains the generation contract

Relevant files:

- [examples/README.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/README.md)
- [examples/provider/provider.tf](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/provider/provider.tf)
- [examples/resources/unifi_network/resource.tf](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/resources/unifi_network/resource.tf)
- [examples/resources/unifi_network/import.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/resources/unifi_network/import.sh)
- [examples/data-sources/unifi_site/data-source.tf](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/data-sources/unifi_site/data-source.tf)
- [templates/index.md.tmpl](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/templates/index.md.tmpl)

### 3. Docs generation workflow added for local development and CI

This PR adds a reproducible docs generation entrypoint plus a dedicated CI workflow.

The new workflow includes:

- `scripts/generate-docs.sh` as the repo-local generator entrypoint
- `make docs-generate` to regenerate markdown
- `make docs-check` to regenerate and fail on drift
- a GitHub Actions docs workflow that runs on pull requests and pushes to `master`

Relevant files:

- [scripts/generate-docs.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/scripts/generate-docs.sh)
- [Makefile](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/Makefile)
- [.github/workflows/docs.yml](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.github/workflows/docs.yml)
- [README.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/README.md)

### 4. Local pre-commit enforcement added for docs drift

The repo-local `pre-commit` config now checks generated docs drift in addition to the existing version-drift validation.

One important workflow detail changed here: the docs check now only diffs actual docs-generation inputs instead of the entire `examples/` tree, so unrelated example files such as `examples/basic-site` do not break the docs job.

Relevant files:

- [.pre-commit-config.yaml](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.pre-commit-config.yaml)
- [Makefile](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/Makefile)

## Validation

Validated locally:

- `make docs-check`
- `go test ./...`
- `terraform fmt -check -recursive examples`
