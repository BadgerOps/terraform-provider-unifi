---
page_title: "Getting started"
description: |-
  Create an API key, configure the provider, and apply a first UniFi Network site configuration.
---

# Getting started

This guide walks through the minimum path from a UniFi Network console to a first successful `terraform apply` with `badgerops/unifi`.

## 1. Create an Integration API key

In the UniFi Network UI:

1. Open **Settings**.
2. Open **Control Plane** → **Integrations** (labels vary slightly by Network version).
3. Enable the Integration API if it is not already on.
4. Create an API key and store it somewhere Terraform can read (environment variable, secret store, or encrypted `tfvars`) — not in git.

The provider authenticates with that API key. It does not use a UniFi local-admin username and password.

## 2. Choose an `api_url`

Set `api_url` to the **base** console URL. The provider appends `/integration` when the path is missing.

| Access path | Example `api_url` |
| --- | --- |
| Local console / UDM | `https://192.0.2.1` or `https://unifi.example.com` |
| UniFi Site Manager cloud connector | `https://api.ui.com/v1/connector/consoles/<console-id>/network` |

Self-signed lab consoles often need `allow_insecure = true`. Prefer real certificates in production.

After WiFi or other changes, UniFi may stop answering the API for minutes while devices reprovision. If applies fail with `context deadline exceeded`, raise `request_timeout_seconds` (default `30`).

## 3. Minimal configuration

```terraform
terraform {
  required_providers {
    unifi = {
      source  = "badgerops/unifi"
      version = "~> 0.5"
    }
  }
}

provider "unifi" {
  api_url = var.api_url
  api_key = var.api_key
  # allow_insecure           = true
  # request_timeout_seconds  = 300
}

data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_network" "lab" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "tf-lab"
  enabled                 = true
  vlan_id                 = 40
  isolation_enabled       = false
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = true

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.40.0.1"
    prefix_length      = 24

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.40.0.100"
      end_ip_address                  = "10.40.0.200"
      lease_time_seconds              = 86400
      dns_server_ip_addresses         = ["1.1.1.1"]
      ping_conflict_detection_enabled = false
    }
  }
}
```

Then:

```bash
terraform init
terraform plan
terraform apply
```

Use a disposable lab VLAN/SSID while you learn the provider. Live acceptance tests in this repository follow the same rule.

## 4. Larger reference example

The repository includes an end-to-end operator example that stands up networks, WiFi, firewall, DNS, and ACL objects:

[examples/basic-site](https://github.com/BadgerOps/terraform-provider-unifi/tree/master/examples/basic-site)

That directory is a reference configuration inside the provider repo, not a published Terraform module.

## 5. Next steps

- [Zone-based firewall](zone-based-firewall) before managing zones or policies
- [Importing existing configuration](importing-existing-configuration) for brownfield sites
- [Compared to legacy UniFi providers](vs-legacy-providers) if you are migrating from `paultyng/unifi` or `ubiquiti-community/unifi`
