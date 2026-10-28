---
page_title: "Compared to legacy UniFi providers"
description: |-
  How badgerops/unifi differs from paultyng/unifi and ubiquiti-community/unifi.
---

# Compared to legacy UniFi providers

Several Terraform providers talk to UniFi. They are not drop-in replacements for each other.

| | `badgerops/unifi` | `paultyng/unifi` / `ubiquiti-community/unifi` |
| --- | --- | --- |
| Primary API | Official **Integration API** (OpenAPI-backed) | Private controller / admin API (and related extensions) |
| Auth | **API key** | Local username/password (some forks also accept API keys / cloud connector) |
| Focus | Durable **configuration** (networks, WiFi, zone firewall, DNS, ACLs, …) | Broad controller surface including older models and some operational objects |
| Site selection | Per-resource `site_id` (usually from `data.unifi_site`) | Often a provider-level `site` argument |
| Resource names / schema | Integration API field names and enums | Legacy SDN model (`purpose`, `dhcp_start`, …) |

## When to use `badgerops/unifi`

- You want Terraform aligned with Ubiquiti's current Integration API
- API-key auth fits your secret management and audit model
- You care about networks, WiFi broadcasts, zone-based firewall, DNS policy, traffic matching lists, and ACL rules as configuration-as-code
- You are starting fresh or can re-model config rather than import every legacy attribute

## When a legacy provider may still fit

- You depend on resources this provider does not model yet (check each provider's Registry docs)
- You must manage a controller that has no Integration API
- You already have a large state file against the legacy schema and cannot remount it yet

## Migration notes

There is no automated state migrator. Treat a move as a **rebuild or selective import**:

1. Inventory what the legacy provider manages and what still matters.
2. Map each object to a `badgerops/unifi` resource (or data source). Names and attributes will differ — for example gateway LAN/VLAN networks use the Integration `unifi_network` model, not `purpose = "corporate"`.
3. Prefer importing existing Integration-API objects over destroy/recreate on production SSIDs and firewall policy. See [Importing existing configuration](importing-existing-configuration).
4. Enable zone-based firewall in the UI before importing or creating zone firewall resources. See [Zone-based firewall](zone-based-firewall).
5. Run plans against a lab site first. WiFi changes can interrupt the API for several minutes; set `request_timeout_seconds` appropriately.

## What this provider intentionally skips

Day-2 controller operations stay out of scope: adoption, client telemetry, analytics, and similar operational actions. Configuration state is the product.

## Related

- [Getting started](getting-started)
- [Provider overview](https://registry.terraform.io/providers/BadgerOps/unifi/latest/docs)
