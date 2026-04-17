# terraform-provider-unifi

Terraform provider for the current UniFi Network integration API.

This repository tracks the committed UniFi Network OpenAPI snapshot and targets the OpenAPI-backed integration endpoints shipped by current UniFi Network releases. It intentionally focuses on durable configuration workflows instead of the older private controller API and operational actions.

This repository is the source for the `badgerops/unifi` Terraform provider. It is not intended to be consumed as a Terraform module package. If BadgerOps publishes reusable Terraform modules built on top of this provider, they should live in separate repositories named with the standard module pattern `terraform-<provider>-<name>`.

## Current scope

- Provider: `badgerops/unifi`
- Data source: `unifi_site`
- Data source: `unifi_device`
- Data source: `unifi_network`
- Data source: `unifi_wifi_broadcast`
- Data source: `unifi_firewall_zone`
- Data source: `unifi_firewall_policy`
- Data source: `unifi_traffic_matching_list`
- Data source: `unifi_dns_policy`
- Data source: `unifi_acl_rule`
- Data source: `unifi_vpn_server`
- Data source: `unifi_site_to_site_vpn_tunnel`
- Data source: `unifi_dpi_application`
- Data source: `unifi_dpi_application_category`
- Data source: `unifi_country`
- Data source: `unifi_radius_profile`
- Data source: `unifi_device_tag`
- Data source: `unifi_wan`
- Data source: `unifi_switch_stack`
- Data source: `unifi_mc_lag_domain`
- Data source: `unifi_lag`
- Resources:
  - `unifi_network`
  - `unifi_wifi_broadcast`
  - `unifi_firewall_zone`
  - `unifi_firewall_policy`
  - `unifi_firewall_policy_ordering`
  - `unifi_traffic_matching_list`
  - `unifi_dns_policy`
  - `unifi_acl_rule`
  - `unifi_acl_rule_ordering`

The implementation focuses on the common documented fields for those resources and keeps translation logic explicit rather than exposing raw JSON passthrough. Firewall policy ordering and ACL rule ordering are managed through dedicated resources because the controller exposes separate ordering endpoints and treats the per-object `index` as read-only state. `unifi_radius_profile`, `unifi_device_tag`, `unifi_wan`, `unifi_switch_stack`, `unifi_mc_lag_domain`, and `unifi_lag` are data sources because the current shipped integration API only exposes read-only endpoints for them.

## Firewall Prerequisite

Zone-based firewall support must be enabled manually in the UniFi Network UI before Terraform can manage:

- `unifi_firewall_zone`
- `unifi_firewall_policy`
- `unifi_firewall_policy_ordering`

If zone-based firewall is not enabled on the target site, the controller returns `api.firewall.zone-based-firewall-not-configured` and the firewall resources/data sources in this provider will not work for that site.

## Known UniFi Firewall Quirks

Current UniFi controller builds have a few behaviors worth planning around:

- Treat the system `Internal` zone as read-only. Moving networks by `zone_id` works, but explicit updates to built-in zone membership may be rejected by the controller.
- Set `allow_return_traffic` explicitly on every `ALLOW` firewall policy. Some controller builds reject filtered `ALLOW` rules when this field is omitted, even if the intended value is `false`.
- `protocol_filter = { type = "NAMED_PROTOCOL" }` is currently only reliable for `ICMP`. For TCP or UDP service policies, prefer `protocol_filter = { type = "PRESET", preset_name = "TCP_UDP" }` combined with a nested destination `port_filter`.
- `unifi_firewall_policy_ordering` can manage an existing zone pair correctly, but some controllers appear more reliable when ordering is imported or aligned after policy creation instead of being created first from a blank state.
- Legacy user-defined firewall policies created outside Terraform may be difficult to import if the controller list endpoints do not expose stable policy IDs.

## OpenAPI Snapshot

The repo includes a committed OpenAPI snapshot under [`internal/openapi/spec`](./internal/openapi/spec) and a generated client/model package under [`internal/openapi/generated`](./internal/openapi/generated).

Current codegen status:

- pinned generator: `oapi-codegen` `v2.6.0`
- committed snapshot: UniFi Network API `10.2.105`
- committed generation scope: full snapshot
- generation inputs:
  - [`internal/openapi/oapi-codegen.yaml`](./internal/openapi/oapi-codegen.yaml)
  - [`internal/openapi/overlay.yaml`](./internal/openapi/overlay.yaml)

Because upstream `oapi-codegen` still does not claim full OpenAPI `3.1` support, the repo keeps the upstream snapshot untouched and applies an OpenAPI Overlay at generation time to downgrade the declared document version to `3.0.3`. The provider code continues to isolate generated DTOs behind explicit translation code in [`internal/translate`](./internal/translate).

## Generated Docs

Schema-driven provider docs are generated into [`docs`](./docs) with `tfplugindocs`.

Generation inputs come from:

- Terraform schema definitions in [`internal/provider`](./internal/provider)
- provider/resource/data source examples under [`examples/provider`](./examples/provider), [`examples/resources`](./examples/resources), and [`examples/data-sources`](./examples/data-sources)
- resource import examples under [`examples/resources`](./examples/resources)
- provider landing-page template under [`templates/index.md.tmpl`](./templates/index.md.tmpl)

The checked-in example contract is documented in [`examples/README.md`](./examples/README.md). Each provider object should have a minimal public-facing example that shows the intended Terraform workflow, not an internal test fixture.

The example directories under [`examples`](./examples) exist to drive generated provider documentation and to provide operator reference configurations. They are examples inside the provider repository, not versioned module entrypoints for Registry consumption.

Generation workflow:

1. Update the schema implementation in [`internal/provider`](./internal/provider).
2. Add or update the matching example files in [`examples`](./examples).
3. Run `make docs-generate` to regenerate `docs/`.
4. Run `make docs-check` to confirm there is no drift in generated docs, templates, or documentation example inputs.
5. Commit both the code changes and the generated markdown.

Useful commands:

```bash
make docs-generate
make docs-check
```

Enforcement:

- CI runs `make docs-check` on pull requests and on pushes to the default branch
- [`.pre-commit-config.yaml`](./.pre-commit-config.yaml) can run the same docs check locally before commit

## Install From The Registry

Released versions are published under `badgerops/unifi` and can be installed directly from the Terraform Registry.

```hcl
terraform {
  required_providers {
    unifi = {
      source = "badgerops/unifi"
      version = "0.2.8"
    }
  }
}

provider "unifi" {
  api_url        = var.api_url
  api_key        = var.api_key
  allow_insecure = var.allow_insecure
}
```

## Local Development Overrides

For local development, use a Terraform CLI development override that points at a directory containing a locally built provider binary:

```hcl
# ~/.terraformrc or $TF_CLI_CONFIG_FILE
provider_installation {
  dev_overrides {
    "badgerops/unifi" = "/absolute/path/to/terraform-provider-unifi"
  }
  direct {}
}
```

Then build the binary in the repo root:

```bash
go build -o terraform-provider-unifi_v0.2.8 .
```

## Filesystem Mirror Installs

Each GitHub release includes:

- platform archives named `terraform-provider-unifi_<version>_<os>_<arch>.zip`
- a Registry manifest asset named `terraform-provider-unifi_<version>_manifest.json`
- `terraform-provider-unifi_<version>_SHA256SUMS` plus the detached signature file `terraform-provider-unifi_<version>_SHA256SUMS.sig`
- a `terraform-provider-unifi_<version>_terraform-mirror.tar.gz` bundle for air-gapped or mirrored Terraform installs

The release workflow reads the version and notes from `CHANGELOG.md`, builds the cross-platform archives, packages the filesystem mirror bundle, and signs the checksum file for Terraform Registry ingestion.

Before the first Registry publish under the `badgerops` namespace:

- add an ASCII-armored public GPG key in the Terraform Registry signing key settings for `badgerops`
- configure the `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets in GitHub Actions

Extract that mirror bundle and point Terraform at it:

```hcl
# terraform.rc
provider_installation {
  filesystem_mirror {
    path    = "/opt/terraform/providers"
    include = ["badgerops/unifi"]
  }
  direct {
    exclude = ["badgerops/unifi"]
  }
}
```

Then in CI:

```bash
export TF_CLI_CONFIG_FILE="$PWD/terraform.rc"
terraform init
terraform plan
```

The filesystem mirror bundle already contains the directory layout Terraform expects under `registry.terraform.io/badgerops/unifi/<version>/<os>_<arch>/`.

## Imports

Resources that belong to a site use composite import IDs:

- `unifi_network`: `<site_id>/<network_id>`
- `unifi_wifi_broadcast`: `<site_id>/<wifi_broadcast_id>`
- `unifi_firewall_zone`: `<site_id>/<firewall_zone_id>`
- `unifi_firewall_policy`: `<site_id>/<firewall_policy_id>`
- `unifi_firewall_policy_ordering`: `<site_id>/<source_zone_id>/<destination_zone_id>`
- `unifi_traffic_matching_list`: `<site_id>/<traffic_matching_list_id>`
- `unifi_dns_policy`: `<site_id>/<dns_policy_id>`
- `unifi_acl_rule`: `<site_id>/<acl_rule_id>`
- `unifi_acl_rule_ordering`: `<site_id>`

## Development

If you use Nix, enter the pinned development shell with:

```bash
nix develop
```

The Nix shell exposes a `terraform` command via an OpenTofu compatibility wrapper for fast local validation, while CI still runs HashiCorp Terraform `1.14.8`.

If you use `direnv`, the flake also provides `direnv` in the dev shell. A simple local setup is:

```bash
# .envrc
use flake

export TF_ACC=1
export UNIFI_API_URL=https://unifi.example.com
export UNIFI_API_KEY=replace-me
export UNIFI_ALLOW_INSECURE=false
export UNIFI_TEST_SITE_NAME=Terraform Acceptance
export UNIFI_TEST_NAME_PREFIX=acctest-
```

Then run:

```bash
direnv allow
```

`make testacc` will use `.envrc` when `direnv` is available and will prefer `/usr/bin/terraform` for acceptance runs so the Terraform plugin test framework does not accidentally pick up the OpenTofu compatibility wrapper from the dev shell.

Useful local commands:

```bash
make fmt
make test
make build
make sync-version
make check-version-drift
make docs-generate
make docs-check
make release-artifacts VERSION=0.2.8
make sign-release-artifacts VERSION=0.2.8
make terraform-fmt-check
make openapi-generate
make testacc
```

If you use `pre-commit`, install the repo hooks with:

```bash
pre-commit install
```

The version-drift hook derives the current release from [`CHANGELOG.md`](./CHANGELOG.md) and updates checked-in references before the commit is finalized.

The [`examples/basic-site`](./examples/basic-site) configuration exercises the provider source address used by the final registry namespace and is validated in CI via a Terraform development override.

### Pull Requests

Every pull request must update [`CHANGELOG.md`](./CHANGELOG.md).

- Bump the version for the next release entry on the branch.
- Add a concise summary of the user-visible changes shipped by the PR.

## Live Acceptance Tests

The repo also includes live controller-backed acceptance tests under `internal/provider`. These are separate from the mock-backed provider tests and only run when `TF_ACC=1` is set.

Recommended local setup:

```bash
cp .env.example .env.testacc
make testacc
```

If you already use `.env` for another tool such as Docker Compose, keep acceptance settings in a separate file or in `.envrc`.

Required environment variables:

- `UNIFI_API_URL`
- `UNIFI_API_KEY`
- exactly one of `UNIFI_TEST_SITE_ID` or `UNIFI_TEST_SITE_NAME`

Optional environment variables:

- `UNIFI_ALLOW_INSECURE`
- `UNIFI_TEST_NAME_PREFIX`
- `UNIFI_TEST_WIFI_PASSPHRASE`
- `UNIFI_TEST_ENABLE_ZONE_FIREWALL`

Use a dedicated disposable UniFi site for these tests. The live suite creates and destroys real resources.

Live test behavior:

- core coverage always runs: `unifi_site`, `unifi_network`, `unifi_traffic_matching_list`, `unifi_dns_policy`, `unifi_acl_rule`
- switch inventory coverage runs when the target site has at least one adopted device with the `switching` feature
- WiFi broadcast coverage is skipped unless `UNIFI_TEST_WIFI_PASSPHRASE` is set; when enabled it also exercises `broadcasting_device_filter` with `DEVICE_TAGS`
- zone firewall coverage is skipped unless `UNIFI_TEST_ENABLE_ZONE_FIREWALL=1` is set
- inventory-backed data sources such as `unifi_wan`, `unifi_radius_profile`, `unifi_device_tag`, `unifi_switch_stack`, `unifi_mc_lag_domain`, and `unifi_lag` skip when the target site has no matching objects
