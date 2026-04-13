# Migration

This provider is a new implementation against the current UniFi integration API. It is not a drop-in continuation of older providers built on the legacy controller API.

## Resource Mapping

Conservative mappings from common older concepts to this provider:

- `unifi_network` -> `unifi_network`
- `unifi_wlan` -> `unifi_wifi_broadcast`
- legacy firewall rule/group patterns -> `unifi_firewall_zone`, `unifi_firewall_policy`, `unifi_traffic_matching_list`, and `unifi_acl_rule`
- DNS override-style records -> `unifi_dns_policy`

Some older resources do not have a one-to-one replacement because the current API model is different. Firewall is the largest example: the current public model is centered on zones, policies, traffic matching lists, and ACL rules instead of a single legacy firewall rule abstraction.

Two migration details matter for current firewall support:

- `unifi_firewall_policy` now uses nested `source_filter` and `destination_filter` blocks instead of older flat network or port attributes.
- Firewall policy order and ACL rule order are managed separately through `unifi_firewall_policy_ordering` and `unifi_acl_rule_ordering`.

## Manual Migration Flow

1. Update configuration to use the `badgerops/unifi` provider source.
2. Rename resources to the new shapes where needed.
3. Import existing UniFi objects into the new state using the current provider resource addresses.
4. Run `terraform plan` and reconcile any schema differences before apply.

## Import Examples

Most site-scoped resources use import IDs in the form `<site_id>/<resource_id>`.

Examples:

```bash
terraform import unifi_network.trusted <site_id>/<network_id>
terraform import unifi_wifi_broadcast.staff <site_id>/<wifi_broadcast_id>
terraform import unifi_firewall_zone.trusted <site_id>/<firewall_zone_id>
terraform import unifi_firewall_policy.trusted_to_iot <site_id>/<firewall_policy_id>
terraform import unifi_firewall_policy_ordering.trusted_to_iot_order <site_id>/<source_zone_id>/<destination_zone_id>
terraform import unifi_traffic_matching_list.web_ports <site_id>/<traffic_matching_list_id>
terraform import unifi_dns_policy.printer <site_id>/<dns_policy_id>
terraform import unifi_acl_rule.block_iot_dns <site_id>/<acl_rule_id>
terraform import unifi_acl_rule_ordering.site_acl_order <site_id>
```

## Discovery Helpers

The provider includes read-only data sources that can help identify current IDs before import:

- `unifi_site`
- `unifi_network`
- `unifi_firewall_zone`
- `unifi_firewall_policy`
- `unifi_traffic_matching_list`
- `unifi_dns_policy`
- `unifi_acl_rule`
- `unifi_vpn_server`
- `unifi_site_to_site_vpn_tunnel`
- `unifi_dpi_application`
- `unifi_dpi_application_category`
- `unifi_country`
- `unifi_radius_profile`
- `unifi_device_tag`
- `unifi_wan`
- `unifi_switch_stack`
- `unifi_mc_lag_domain`
- `unifi_lag`

Switching and WAN are data-source only in the current provider because the shipped `10.2.105` integration API exposes them as read-only endpoints.
