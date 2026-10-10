# Changelog

All notable changes to this project will be documented in this file.

The format follows Keep a Changelog and the release numbers follow Semantic Versioning.

## [Unreleased]

### Added

- Added official Integration API support for WPA2 Enterprise, mixed WPA2/WPA3 Enterprise, and WPA3 Enterprise WiFi security, including RADIUS profiles, NAS-ID, MAC authentication, Change of Authorization, roaming, PMF, rekeying, and WPA3 security mode.
- Added WPA2 Personal PPSKs with per-key network assignment, sensitive state handling, and refresh preservation for passphrases omitted by UniFi responses. A PPSK broadcast omits the broadcast-level `network`, because each key carries its own.

### Fixed

- Made `network` optional on `unifi_wifi_broadcast`. The controller forbids a broadcast-level network alongside `security_configuration.preshared_keys`, because each preshared key carries its own, so PPSK broadcasts could not be created at all while the attribute was required. It is still required for every other configuration, and the integration schema never listed it as required.
- Rejected `network` and `security_configuration.passphrase` alongside `security_configuration.preshared_keys` at plan time, matching the controller.
- Rejected `security_configuration.pmf_mode`, `security_configuration.fast_roaming_enabled`, and `security_configuration.group_rekey_interval_seconds` at plan time for `IOT_OPTIMIZED` broadcasts. The integration API documents all three as unavailable for IoT configuration, for every security type, so the controller silently ignored them.

## [0.4.0] - 2026-10-06

### Added

- Added `unifi_port_forward` and `data.unifi_port_forward` for WAN port forwarding rules, using the legacy UniFi Network `rest/portforward` endpoint because the integration API does not expose port forwarding. Rules support create, read, update, delete, and `<site_id>/<id>` import; the data source selects by legacy ID or unique name. Port attributes are strings so single ports, ranges, and comma-separated lists round-trip without normalisation.
- `source` restricts a rule to an IPv4 address or CIDR prefix, or `any`. It reads back as `any` when source limiting is disabled on the controller, so a restriction switched off outside Terraform shows as a diff and is restored on apply.
- `source`, `destination_port`, `forward_port`, and `forward_ip` are validated at plan time.

### Changed

- Port forward updates send the complete controller object with only the modelled fields overlaid, so destination IP filters, source firewall groups, and fields the provider does not expose survive the legacy API's full-replace `PUT`. Setting a specific `source` on a rule that uses a source firewall group is rejected, because the controller treats the two as mutually exclusive.
- The legacy API helper shared with `unifi_dhcp_reservation` now reads the `meta` envelope on every response, so a failure reported with HTTP `200` is an error rather than a success, and `api.err.IdInvalid` and `api.err.NotFound` are treated as not found so deleting a rule that was already removed on the controller succeeds.
- Client error diagnostics include the controller response body when it carries more detail than the code and message, so legacy validation errors name the rejected field.

## [0.3.1] - 2026-10-04

### Fixed

- Fixed the weekly OpenAPI upstream check, which had failed on every run since 2026-08-03. It read the `stable` apt package index, which Ubiquiti left empty when that suite moved to the `10.5` line. The check now asks the Ubiquiti firmware API for the latest `unifi-native` release package and verifies the download against the published checksum.
- The upstream check workflow now opens a tracking issue when the check itself fails, instead of only failing the scheduled run.
- `data.unifi_switch_stack` keeps returning `member_device_ids` on controllers older than `10.6`. The client now decodes switch stack pages from the raw response, because the regenerated DTO no longer carries the `members` field those controllers return.

### Changed

- Refreshed the committed UniFi Network integration OpenAPI snapshot from `10.3.58` to `10.6.106` and regenerated the pinned `oapi-codegen` client. The `10.6.106` document adds no endpoints.
- Added optional `channel_2g_locked_to_6` and `dtim_period_2g_locked_to_3` attributes to `unifi_wifi_broadcast` and its data source, for the WiFi broadcast fields that UniFi Network `10.6` introduced. They are only sent when set, so existing configurations and older controllers are unaffected.
- `data.unifi_switch_stack` now exposes `device_id` and `unit_mac_addresses`. UniFi Network `10.6` reports stack members as units keyed by MAC address instead of device ID, so `member_device_ids` is resolved from the site's adopted devices on those controllers.
- `data.unifi_lag` resolves `member_device_ids` for switch stack LAGs on UniFi Network `10.6`, which identifies those members by unit MAC address.
- `scripts/check-openapi-upstream.sh` accepts `--save-spec PATH` to write the extracted `api-docs/integration.json`, and takes `--firmware-api-url`, `--product`, `--channel` and `--platform` in place of the removed apt options.

## [0.3.0] - 2026-10-04

### Security

- Bumped `golang.org/x/crypto` from `0.48.0` to `0.55.0`, resolving the open Dependabot advisories fixed in `0.52.0` for the SSH and SSH agent packages (including GHSA-5cgq-3rg8-m6cv, GHSA-jppx-rxg9-jmrx, GHSA-f5wc-c3c7-36mc, GHSA-89gr-r52h-f8rx, GHSA-x527-x647-q7gg, GHSA-vgwf-h737-ff37, and GHSA-rm3j-f69w-wqmq).
- Bumped `google.golang.org/grpc` from `1.79.3` to `1.83.2`, resolving GHSA-hrxh-6v49-42gf, GHSA-vp52-pcj8-j9qc, GHSA-qc2q-p7wx-3px3, and GHSA-2v4p-qf9q-27wj (xDS RBAC bypass, HTTP/2 DATA frame memory exhaustion, and a crash on requests missing `:authority` and `Host` headers).
- Bumped `golang.org/x/net` from `0.49.0` to `0.58.0`, resolving GHSA-5cv4-jp36-h3mw (HTML parser denial of service).

### Changed

- Updated the transitive `golang.org/x/mod`, `golang.org/x/sync`, `golang.org/x/sys`, `golang.org/x/text`, `golang.org/x/tools`, and `google.golang.org/genproto/googleapis/rpc` modules pulled in by the dependency bumps above.
- Promoted `github.com/google/uuid` to a direct requirement in `go.mod` to match its existing direct use in the provider.

## [0.2.15] - 2026-06-29

### Fixed

- Preserved UniFi Network metadata when translating generated network details into the provider client model.

## [0.2.14] - 2026-06-06

### Fixed

- Fixed the weekly OpenAPI upstream check so application package-only version bumps do not open provider snapshot drift issues when the upstream artifact does not include a newer `api-docs/integration.json`.

## [0.2.13] - 2026-05-25

### Fixed

- Fixed legacy DHCP reservation API URL derivation when `api_url` already ends in `/proxy/network/integration`.
- Fixed `unifi_dhcp_reservation` refresh behavior so missing legacy client records are removed from Terraform state and can be recreated on the next apply.

## [0.2.12] - 2026-05-05

### Changed

- Refreshed the committed UniFi Network integration OpenAPI snapshot from `10.2.105` to `10.3.58` and regenerated the pinned `oapi-codegen` client.
- Updated OpenAPI snapshot metadata and documentation to track the `unifi-native` `10.3.58-34147-1` source package and snapshot checksum.
- Added provider support for UniFi Network `10.3.58` WiFi schema additions: open security encryption modes and standard-broadcast DNS assistance configuration.
- Kept `mdns_forwarding_enabled` explicitly managed for gateway networks even though the refreshed upstream schema now allows the controller site default when the field is omitted.

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
