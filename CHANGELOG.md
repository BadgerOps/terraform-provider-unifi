# Changelog

All notable changes to this project will be documented in this file.

The format follows Keep a Changelog and the release numbers follow Semantic Versioning.

## [0.2.11] - 2026-04-20

### Fixed

- `unifi_dhcp_reservation` now auto-creates the missing legacy configured-client record for adopted UniFi infrastructure devices before applying the reservation, while retaining coverage for both pre-existing client records and the adopted-device bootstrap path.

## [0.2.10] - 2026-04-20

### Added

- Added `unifi_dhcp_reservation` for managing DHCP reservations by site and MAC address, using the legacy UniFi Network client database endpoint because the committed integration API snapshot does not expose DHCP reservation writes.

### Changed

- Extended the client to derive both the integration API base URL and the legacy `/proxy/network/api` base URL from the configured `api_url` without breaking existing `/proxy/network` and `/integration` input shapes.
- Added mock-backed provider coverage for DHCP reservation CRUD/import behavior and for `unifi_firewall_policy` rules that use `action = "ALLOW"`, `allow_return_traffic = true`, and a destination `NETWORK` filter.
- Generated docs and examples now document `unifi_dhcp_reservation` and call out that it is the current exception to the provider's otherwise integration-API-focused surface.

## [0.2.9] - 2026-04-19

### Changed

- Reworked the Terraform Registry provider overview so the generated `docs/index.md` leads with public-facing guidance on what the provider manages, the required UniFi integration API setup, and the main configuration prerequisites.
- Pinned GitHub Actions Terraform setup to `1.5.7` and updated the Go test workflow to run provider tests against the installed Terraform binary instead of auto-downloading a CLI during test execution.

## [0.2.8] - 2026-04-17

### Changed

- Fixed release checksum generation so `terraform-provider-unifi_<version>_SHA256SUMS` records zip asset names without a leading `./`, matching the filenames attached to GitHub releases and accepted by Terraform Registry ingestion.

## [0.2.7] - 2026-04-17

### Changed

- Clarified the repository structure so the provider repo is explicitly documented as a provider distribution repository, not a reusable module repository.
- Added local READMEs for `examples/provider` and `examples/basic-site` so Registry-discovered nested Terraform directories are clearly described as examples rather than undocumented internal-only submodules.
- Normalized generated `docs/index.md` EOF handling in `scripts/generate-docs.sh` so `make docs-check` does not keep failing on a trailing newline drift in CI.

## [0.2.6] - 2026-04-17

### Changed

- Terraform Registry release artifacts now include the versioned provider manifest and a signed `SHA256SUMS` file so GitHub releases are ready for Registry ingestion.
- GitHub release automation now requires the `GPG_PRIVATE_KEY` and `PASSPHRASE` repository secrets and signs the published checksum file during release creation.
- README release guidance now describes the Registry-compatible release assets and the extra setup needed before the first publish under the `badgerops` namespace.

## [0.2.5] - 2026-04-17

### Changed

- Renamed internal client and translation files so the repository no longer carries `phase*` or `*_spike` implementation artifacts.
- Reworked the release-facing docs to lead with public registry usage and describe local development overrides and filesystem mirror installs without internal-only wording.
- Updated docs generation to export schema through the published `badgerops/unifi` source address instead of relying on `tfplugindocs`' `hashicorp/<name>` default.

## [0.2.4] - 2026-04-15

### Changed

- `unifi_firewall_policy` now requires `allow_return_traffic` to be set explicitly when `action = "ALLOW"` so Terraform catches a controller requirement that previously surfaced only as an API error during apply.
- `unifi_firewall_policy` now rejects `protocol_filter = { type = "NAMED_PROTOCOL", named_protocol = "tcp" }` and similar TCP/UDP variants because current UniFi controller builds only reliably accept `ICMP` for `NAMED_PROTOCOL`.

### Fixed

- Updated firewall policy examples, docs, and acceptance coverage to use the controller-safe `PRESET/TCP_UDP` protocol filter pattern for port-scoped TCP/UDP service rules.
- Documented live-controller firewall quirks around built-in zones, explicit `allow_return_traffic`, protocol filters, and policy ordering/import behavior.

## [0.2.3] - 2026-04-13

### Added

- Generated provider documentation under `docs/`, driven by Terraform schema plus checked-in provider, resource, data source, and import examples.
- `make docs-generate` and `make docs-check` for reproducible documentation generation and validation.
- GitHub Actions docs workflow to detect drift in generated docs, templates, and documentation examples.
- Checked-in docs generation inputs and local enforcement via `templates/index.md.tmpl`, `examples/README.md`, and a `pre-commit` docs drift hook.

## [0.2.2] - 2026-04-13

### Added

- `unifi_wifi_broadcast` data source for looking up WiFi broadcasts by `id` or `name` within a site.

## [0.2.1] - 2026-04-13

### Fixed

- Removed the temporary legacy-provider migration document and updated the repository docs to align with the shared BadgerOps plan and the committed UniFi OpenAPI snapshot as the source of truth.
- Added repo-local pre-commit and CI version-drift checks so README examples, the Terraform example configuration, and local validation wiring stay in sync with the current `CHANGELOG.md` release version.
- Updated Terraform example validation to derive the local mirror version from `CHANGELOG.md` instead of a hardcoded provider version.

## [0.2.0] - 2026-04-13

### Added

- Firewall policy ordering via `unifi_firewall_policy_ordering`, including support for controller ordering before and after system-defined rules.
- ACL rule ordering via `unifi_acl_rule_ordering`.
- Firewall reference data sources: `unifi_vpn_server`, `unifi_site_to_site_vpn_tunnel`, `unifi_dpi_application`, `unifi_dpi_application_category`, and `unifi_country`.
- Read support for `unifi_firewall_policy`, `unifi_dns_policy`, and `unifi_acl_rule` data sources.
- Expanded traffic matching list support for IPv4 and IPv6 address entries in addition to ports.

### Changed

- `unifi_firewall_policy` now models the full nested `source_filter` and `destination_filter` structure used by the current UniFi integration API, including port, network, MAC, IP, IPv6 IID, region, VPN, domain, DPI application, and DPI category selectors.
- Mock Terraform provider tests now run in the default `go test` path instead of being hidden behind `TF_ACC`, substantially increasing normal provider test coverage.

## [0.1.1] - 2026-04-13

### Fixed

- Network create and update requests now use raw JSON over the generated transport so required fields like `isolationEnabled`, `internetAccessEnabled`, `mdnsForwardingEnabled`, `ipv4Configuration`, and `cellularBackupEnabled` are preserved.
- Fixed imported network management under Terraform `dev_overrides`, where apply could fail with `api.request.error` because the generated OpenAPI request type dropped required update payload fields.

## [0.1.0] - 2026-04-12

### Added

- Release automation for internal provider consumption, including GitHub release assets and a Terraform filesystem mirror bundle.
- Cross-platform packaging via `scripts/build-release-artifacts.sh` and `make release-artifacts VERSION=...`.
- README guidance for local `dev_overrides` installs and CI `filesystem_mirror` installs.

### Changed

- Release publishing now derives the version and notes from this changelog and runs when changes land on `master`.

## [0.0.1] - 2026-04-12

### Added

- Initial UniFi provider implementation for the OpenAPI-backed integration API.
- Resources: `unifi_network`, `unifi_wifi_broadcast`, `unifi_firewall_zone`, `unifi_firewall_policy`, `unifi_traffic_matching_list`, `unifi_dns_policy`, `unifi_acl_rule`.
- Data sources: `unifi_site`, `unifi_device`, `unifi_network`, `unifi_firewall_zone`, `unifi_traffic_matching_list`, `unifi_radius_profile`, `unifi_device_tag`, `unifi_wan`, `unifi_switch_stack`, `unifi_mc_lag_domain`, `unifi_lag`.
- Generated OpenAPI client integration and translation boundary.
- Mock-backed and live controller-backed acceptance coverage.
- WiFi `broadcasting_device_filter` support for `DEVICE_TAGS`.
- Nix development shell, Terraform example configuration, and CI validation workflows.
