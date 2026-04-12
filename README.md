# terraform-provider-unifi

Terraform provider for the current UniFi Network integration API.

This repository implements the new provider shape described in the adjacent BadgerOps design docs. It targets the OpenAPI-backed integration endpoints shipped by current UniFi Network releases instead of the older private controller API.

## Current scope

- Provider: `badgerops/unifi`
- Data source: `unifi_site`
- Data source: `unifi_network`
- Data source: `unifi_firewall_zone`
- Resources:
  - `unifi_network`
  - `unifi_wifi_broadcast`
  - `unifi_firewall_zone`
  - `unifi_firewall_policy`

The initial implementation focuses on the common documented fields for those resources and keeps translation logic explicit rather than exposing raw JSON passthrough.

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

## Development

If you use Nix, enter the pinned development shell with:

```bash
nix develop
```

The Nix shell exposes a `terraform` command via an OpenTofu compatibility wrapper for fast local validation, while CI still runs HashiCorp Terraform `1.14.8`.

Useful local commands:

```bash
make fmt
make test
make build
make terraform-fmt-check
```

The [`examples/basic-site`](./examples/basic-site) configuration exercises the provider source address used by the final registry namespace and is validated in CI via a Terraform development override.
