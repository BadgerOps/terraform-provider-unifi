---
page_title: "Zone-based firewall"
description: |-
  Enable zone-based firewall in the UniFi UI and plan around known controller quirks.
---

# Zone-based firewall

Zone-based firewall support must be enabled **manually in the UniFi Network UI** before Terraform can manage:

- [`unifi_firewall_zone`](../resources/firewall_zone.md)
- [`unifi_firewall_policy`](../resources/firewall_policy.md)
- [`unifi_firewall_policy_ordering`](../resources/firewall_policy_ordering.md)

If it is not enabled on the target site, the controller returns `api.firewall.zone-based-firewall-not-configured` and these resources and data sources will not work for that site.

## Known UniFi firewall quirks

Current UniFi controller builds have behaviors worth planning around:

- Treat the system **Internal** zone as read-only. Moving networks by `zone_id` works, but explicit updates to built-in zone membership may be rejected by the controller.
- Set `allow_return_traffic` **explicitly** on every `ALLOW` firewall policy. Some controller builds reject filtered `ALLOW` rules when this field is omitted, even if the intended value is `false`.
- `protocol_filter = { type = "NAMED_PROTOCOL" }` is currently only reliable for `ICMP`. For TCP or UDP service policies, prefer `protocol_filter = { type = "PRESET", preset_name = "TCP_UDP" }` combined with a nested destination `port_filter`.
- `unifi_firewall_policy_ordering` can manage an existing zone pair correctly, but some controllers are more reliable when ordering is **imported or aligned after policy creation** instead of being created first from a blank state.
- Legacy user-defined firewall policies created outside Terraform may be difficult to import if the controller list endpoints do not expose stable policy IDs.

## Ordering resources

Firewall policy order and ACL rule order are separate controller endpoints. The per-object `index` field is read-only state, so this provider manages order through:

- [`unifi_firewall_policy_ordering`](../resources/firewall_policy_ordering.md)
- [`unifi_acl_rule_ordering`](../resources/acl_rule_ordering.md)

Import IDs for those resources are documented in [Importing existing configuration](./importing-existing-configuration.md).

## Related

- [Getting started](./getting-started.md)
- [Compared to legacy UniFi providers](./vs-legacy-providers.md)
