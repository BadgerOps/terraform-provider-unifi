---
page_title: "Importing existing configuration"
description: |-
  Composite import IDs and brownfield tips for badgerops/unifi resources.
---

# Importing existing configuration

Most managed objects are site-scoped. Import IDs are **composite** strings, not bare UniFi object IDs.

## Import ID formats

| Resource | Import ID |
| --- | --- |
| `unifi_network` | `<site_id>/<network_id>` |
| `unifi_wifi_broadcast` | `<site_id>/<wifi_broadcast_id>` |
| `unifi_firewall_zone` | `<site_id>/<firewall_zone_id>` |
| `unifi_firewall_policy` | `<site_id>/<firewall_policy_id>` |
| `unifi_firewall_policy_ordering` | `<site_id>/<source_zone_id>/<destination_zone_id>` |
| `unifi_traffic_matching_list` | `<site_id>/<traffic_matching_list_id>` |
| `unifi_dns_policy` | `<site_id>/<dns_policy_id>` |
| `unifi_acl_rule` | `<site_id>/<acl_rule_id>` |
| `unifi_acl_rule_ordering` | `<site_id>` |
| `unifi_dhcp_reservation` | `<site_id>/<mac_address>` (legacy API) |
| `unifi_port_forward` | `<site_id>/<id>` (legacy API) |

Example:

```bash
terraform import unifi_network.trusted <site_id>/<network_id>
```

Each resource page also includes a generated import snippet under **Import**.

## Brownfield tips

1. Resolve the site first with [`data.unifi_site`](../data-sources/site) (by name) and use that `id` in every import string and resource.
2. Import networks and zones before policies that reference them; import policies before ordering resources when possible.
3. For WiFi PPSK broadcasts, UniFi responses may omit passphrases. After import, supply passphrases in configuration so state stays complete — see [`unifi_wifi_broadcast`](../resources/wifi_broadcast).
4. Enable zone-based firewall in the UI before importing firewall zones or policies. See [Zone-based firewall](zone-based-firewall).
5. Expect a first plan after import to show drift on fields the UI left unset or that the Integration API normalizes (booleans, enums, frequency lists). Prefer aligning config to the API over force-replacing production objects.

## Related

- [Getting started](getting-started)
- [Zone-based firewall](zone-based-firewall)
