# terraform-provider-unifi

Terraform provider for the current UniFi Network integration API.

This repository implements the new provider shape described in the adjacent BadgerOps design docs. It targets the OpenAPI-backed integration endpoints shipped by current UniFi Network releases instead of the older private controller API.

## Current scope

- Provider: `badgerops/unifi`
- Data source: `unifi_site`
- Data source: `unifi_network`
- Data source: `unifi_firewall_zone`
- Data source: `unifi_traffic_matching_list`
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
  - `unifi_traffic_matching_list`
  - `unifi_dns_policy`
  - `unifi_acl_rule`

The implementation focuses on the common documented fields for those resources and keeps translation logic explicit rather than exposing raw JSON passthrough. `unifi_radius_profile`, `unifi_device_tag`, `unifi_wan`, `unifi_switch_stack`, `unifi_mc_lag_domain`, and `unifi_lag` are data sources because the current shipped integration API only exposes read-only endpoints for them.

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

## Provider example

```hcl
terraform {
  required_providers {
    unifi = {
      source = "badgerops/unifi"
    }
  }
}

provider "unifi" {
  api_url        = var.api_url
  api_key        = var.api_key
  allow_insecure = var.allow_insecure
}
```

## Imports

Resources that belong to a site use composite import IDs:

- `unifi_network`: `<site_id>/<network_id>`
- `unifi_wifi_broadcast`: `<site_id>/<wifi_broadcast_id>`
- `unifi_firewall_zone`: `<site_id>/<firewall_zone_id>`
- `unifi_firewall_policy`: `<site_id>/<firewall_policy_id>`
- `unifi_traffic_matching_list`: `<site_id>/<traffic_matching_list_id>`
- `unifi_dns_policy`: `<site_id>/<dns_policy_id>`
- `unifi_acl_rule`: `<site_id>/<acl_rule_id>`

## Migration

Migration guidance for users coming from older UniFi Terraform providers is in [`docs/MIGRATION.md`](./docs/MIGRATION.md).

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

Useful local commands:

```bash
make fmt
make test
make build
make terraform-fmt-check
make openapi-generate
make testacc
```

The [`examples/basic-site`](./examples/basic-site) configuration exercises the provider source address used by the final registry namespace and is validated in CI via a Terraform development override.

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

Use a dedicated disposable UniFi site for these tests. The live suite creates and destroys real resources.
