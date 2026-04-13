# Changelog

All notable changes to this project will be documented in this file.

The format follows Keep a Changelog and the release numbers follow Semantic Versioning.

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
