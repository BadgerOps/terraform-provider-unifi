# PR: Add DHCP Reservation Resource And Expand Firewall Coverage

## Summary

This PR adds `unifi_dhcp_reservation`, using the legacy local UniFi Network client database endpoint for DHCP reservation writes while keeping the rest of the provider integration-first. It also expands firewall-policy test coverage for the `ALLOW + allow_return_traffic = true + destination NETWORK` rule shape, and updates changelog/docs/examples to match.

## Why

The committed UniFi Network OpenAPI snapshot (`10.2.105`) does not expose DHCP reservation writes in the integration API, so there was no supported way to manage reservations through Terraform even though the controller accepts them through the older Network API surface.

Before this PR:

- the provider had no DHCP reservation resource
- the client only modeled the integration API base URL
- docs still described the provider as integration-only with no exception
- firewall tests did not explicitly cover `ALLOW` policies with return traffic enabled and a destination `NETWORK` filter

This PR addresses that by:

- adding a narrow handwritten legacy client for DHCP reservation read/update behavior
- registering a Terraform resource that maps reservation state to `site_id + mac_address`
- documenting the legacy exception clearly in the README and generated docs
- adding mock-backed tests for both the new resource and the missing firewall rule shape

## Implementation

### 1. Add a legacy DHCP reservation client path

The provider client now derives both:

- the integration API base URL
- the legacy `/proxy/network/api` base URL

without breaking existing `api_url` inputs such as:

- `https://controller.example.com`
- `https://controller.example.com/proxy/network`
- `https://controller.example.com/integration`

The new DHCP reservation client:

- resolves site UUID to legacy `internalReference`
- lists legacy client records from `GET /proxy/network/api/s/{site_ref}/rest/user`
- finds clients by MAC address
- updates reservations through sparse `PUT` payloads containing `_id`, `fixed_ip`, and `use_fixedip`
- returns a retry-friendly error if the target MAC is not yet present in the controller client database

Relevant files:

- [internal/client/client.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/client.go)
- [internal/client/dhcp_reservations.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/dhcp_reservations.go)
- [internal/client/errors.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/errors.go)
- [internal/client/client_test.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/client_test.go)

### 2. Add `unifi_dhcp_reservation`

The new resource is intentionally minimal:

- `site_id` required
- `mac_address` required
- `fixed_ip` required
- `enabled` optional, default `true`
- `id` computed as `<site_id>/<mac_address>`

Behavior:

- create/update: upsert the reservation by MAC
- read: preserve present-but-disabled reservations in state
- delete: disable the reservation by setting `use_fixedip = false`
- import: `<site_id>/<mac_address>`

Relevant files:

- [internal/provider/resource_dhcp_reservation.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/resource_dhcp_reservation.go)
- [internal/provider/provider.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/provider.go)

### 3. Expand mock API and provider coverage

The mock UniFi test server now serves both:

- integration API paths under `/integration`
- legacy DHCP reservation paths under `/proxy/network/api/s/{site_ref}/rest/user`

Added coverage includes:

- DHCP reservation CRUD/import behavior
- the firewall policy gap for:
  - `action = "ALLOW"`
  - `allow_return_traffic = true`
  - `source_filter.type = "NETWORK"`
  - `destination_filter.type = "NETWORK"`

Relevant file:

- [internal/provider/provider_test.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/provider_test.go)

## Docs

This PR updates release-facing docs and examples so the new surface is visible outside the code:

- adds an unreleased changelog entry
- updates the provider landing-page template and generated docs index
- adds a checked-in example and import snippet for `unifi_dhcp_reservation`
- generates `docs/resources/dhcp_reservation.md`
- clarifies that DHCP reservations are the current exception to the provider's integration-first model

Relevant files:

- [CHANGELOG.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/CHANGELOG.md)
- [README.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/README.md)
- [templates/index.md.tmpl](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/templates/index.md.tmpl)
- [docs/index.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/index.md)
- [docs/resources/dhcp_reservation.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/docs/resources/dhcp_reservation.md)
- [examples/resources/unifi_dhcp_reservation/resource.tf](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/resources/unifi_dhcp_reservation/resource.tf)
- [examples/resources/unifi_dhcp_reservation/import.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/examples/resources/unifi_dhcp_reservation/import.sh)

## Constraints And Follow-Ups

This PR intentionally does not try to create reservations for devices that are completely absent from the controller's client database. For now:

- if the MAC is not present in `rest/user`, the provider returns a retry-friendly error
- the resource remains a narrow legacy exception rather than broadening the whole provider to legacy/private API behavior
- controller metadata such as hostname, last IP, and network name stays out of Terraform state

Future work, if UniFi exposes official reservation writes in the integration API, should move this resource back onto the supported API surface.

## Validation

Validated locally:

- `env GOPROXY=off GOSUMDB=off CGO_ENABLED=0 TMPDIR=$PWD/.cache/go-tmp GOTMPDIR=$PWD/.cache/go-tmp GOCACHE=$PWD/.cache/go-build GOMODCACHE=/home/badger/go/pkg/mod go test ./internal/client`
- `env GOPROXY=off GOSUMDB=off CGO_ENABLED=0 TMPDIR=$PWD/.cache/go-tmp GOTMPDIR=$PWD/.cache/go-tmp GOCACHE=$PWD/.cache/go-build GOMODCACHE=/home/badger/go/pkg/mod go test ./internal/provider`
- docs regenerated through the manual equivalent of `scripts/generate-docs.sh` inside `nix develop`, pinned to `/usr/bin/terraform`, using cached `terraform-plugin-docs` source
