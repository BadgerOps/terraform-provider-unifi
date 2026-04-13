# PR: Migrate provider client to generated transport and add live acceptance coverage

## Summary

This PR finishes the first generated-client migration pass on top of the overlay-based OpenAPI client and adds live controller-backed acceptance coverage for the implemented provider surface.

It also adds a real adopted-device inventory data source so switch inventory can be queried directly from the Integration API without relying on switch-stack topology endpoints.

## Why

Before this PR:

- the provider had moved to a generated OpenAPI client, but several endpoints still depended on handwritten transport or lossy generated typed models
- acceptance coverage was mostly mock-backed, with only limited live validation
- switch inventory in a real site could not be queried directly, even when switching topology endpoints such as switch stacks or LAGs were empty
- running live tests inside the Nix shell required too much manual environment setup

This PR addresses those gaps by:

- moving the current provider operations onto generated transport while preserving explicit Terraform-facing models
- adding live acceptance coverage that catches real controller behavior and schema mismatches
- adding `unifi_device` for adopted device inventory lookups, including switch discovery via `required_feature = "switching"`
- standardizing the live test workflow under `make testacc`

## Main Changes

### 1. Generated client migration completed for current live-tested endpoints

The remaining lossy endpoints were moved off handwritten HTTP fallback and onto generated request builders with explicit raw JSON request/response handling:

- `traffic_matching_list`
- `dns_policy`
- `acl_rule`

This keeps transport generated while avoiding dependence on broken generated DTOs where upstream schema generation drops fields such as:

- traffic matching list `items`
- DNS policy type-specific fields
- ACL rule polymorphic filters

Relevant files:

- [internal/client/client.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/client.go)
- [internal/client/openapi_helpers.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/openapi_helpers.go)
- [internal/client/resources.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/resources.go)
- [internal/client/phase2.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/phase2.go)

### 2. Live acceptance suite added and expanded

This PR adds a live controller-backed acceptance test suite under:

- [internal/provider/acceptance_live_test.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/acceptance_live_test.go)

Coverage now includes:

- `unifi_site` data source
- `unifi_network` resource and data source
- `unifi_traffic_matching_list` resource and data source
- `unifi_dns_policy` resource
- `unifi_acl_rule` resource
- `unifi_radius_profile` data source
- `unifi_device_tag` data source
- `unifi_wan` data source
- `unifi_device` data source for adopted devices with feature filtering

Optional/gated live coverage is also included for:

- `unifi_wifi_broadcast`
- `unifi_firewall_zone`
- `unifi_firewall_policy`

The live suite:

- requires `TF_ACC=1`
- requires `UNIFI_API_URL` and `UNIFI_API_KEY`
- requires exactly one of `UNIFI_TEST_SITE_ID` or `UNIFI_TEST_SITE_NAME`
- skips WiFi tests unless `UNIFI_TEST_WIFI_PASSPHRASE` is set
- skips zone-firewall tests unless `UNIFI_TEST_ENABLE_ZONE_FIREWALL=1` is set
- skips inventory-backed data source tests when the target site has no matching objects

### 3. New adopted device inventory data source

This PR adds:

- `data "unifi_device"`

It is backed by the adopted device overview endpoint:

- `GET /v1/sites/{siteId}/devices`

Lookup selectors:

- `id`
- `name`
- `mac_address`

Optional filter:

- `required_feature`

This makes switch discovery practical in real environments:

```hcl
data "unifi_device" "switch" {
  site_id          = data.unifi_site.target.id
  name             = "core-switch-a"
  required_feature = "switching"
}
```

Computed fields include:

- `model`
- `ip_address`
- `state`
- `supported`
- `firmware_updatable`
- `firmware_version`
- `features`
- `interfaces`

Relevant files:

- [internal/client/devices.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/client/devices.go)
- [internal/provider/data_source_device.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/data_source_device.go)
- [internal/provider/provider.go](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/internal/provider/provider.go)

### 4. Live test workflow improved for Nix + direnv

This PR also improves the day-to-day test flow:

- adds `direnv` to the Nix shell
- adds a `make testacc` wrapper via [scripts/testacc.sh](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/scripts/testacc.sh)
- makes the wrapper automatically use `.envrc` when `direnv` is present
- prefers `/usr/bin/terraform` so Terraform plugin testing does not accidentally use the OpenTofu compatibility wrapper from the dev shell
- documents the expected `.envrc` / test environment behavior in [README.md](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/README.md) and [.env.example](/home/badger/code/badgerops-unifi-provider/terraform-provider-unifi/.env.example)

## Notable Fixes Found By Live Testing

The live acceptance work exposed and fixed several real issues that mock-only coverage had not caught:

- update handlers were reading computed `id` values from plan instead of prior state
- traffic matching list generated DTOs dropped `items`
- DNS policy generated DTOs dropped type-specific request fields
- ACL rule generated DTOs were too lossy for reliable request/response mapping
- acceptance resource names could exceed controller limits

## Validation

Validated locally:

- `go test ./...`
- `go build ./...`

Validated inside `nix develop`:

- `golangci-lint run ./...`
- `actionlint`
- `terraform fmt -check -recursive examples`
- `tflint --init`
- `tflint --chdir=examples/basic-site`
- `make testacc`

Live controller-backed tests passed for the currently enabled environment on the target site, including:

- `TestAccLiveSiteDataSource`
- `TestAccLiveResourceNetwork`
- `TestAccLiveDataSourceNetwork`
- `TestAccLiveResourceTrafficMatchingList`
- `TestAccLiveDataSourceTrafficMatchingList`
- `TestAccLiveResourceDNSPolicy`
- `TestAccLiveResourceACLRule`
- `TestAccLiveDataSourceRadiusProfile`
- `TestAccLiveDataSourceDeviceTag`
- `TestAccLiveDataSourceWAN`
- `TestAccLiveDataSourceSwitchDevice`

Expected live skips remained in place for:

- WiFi broadcasts when no test passphrase is configured
- zone-firewall resources when zone firewall is not enabled for the site
- switch topology resources when the target site has no switch stacks, LAGs, or MC-LAG domains

## Commit Breakdown

- `04c79b1` `Migrate client transport to generated OpenAPI client`
- `9bb6492` `Add live acceptance test scaffolding`
- `5473d8b` `Add direnv to Nix dev shell`
- `a0b5482` `Bound live acceptance names to UniFi limits`
- `1024cf9` `Use state IDs for resource updates`
- `2193040` `Improve live acceptance workflow and coverage`
- `182c1df` `Finish generated client migration and expand live acceptance`
- `785d922` `Add device inventory data source for switches`

## Follow-ups

Follow-up work that now makes sense on top of this PR:

- add controller capability detection so optional live tests do not rely only on env flags
- add more device-oriented data sources if needed, for example explicit switch-specific views layered on top of `unifi_device`
- implement release tracking automation from the project plan
- continue reducing schema drift where generated DTOs are still lossy relative to the live Integration API
