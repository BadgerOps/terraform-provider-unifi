# Manage a firewall policy between source and destination firewall zones.
data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_network" "trusted" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "trusted"
  enabled                 = true
  vlan_id                 = 20
  isolation_enabled       = false
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = true

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.20.0.1"
    prefix_length      = 24
  }
}

resource "unifi_network" "iot" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "iot"
  enabled                 = true
  vlan_id                 = 30
  isolation_enabled       = true
  cellular_backup_enabled = false
  internet_access_enabled = true
  mdns_forwarding_enabled = false

  ipv4_configuration = {
    auto_scale_enabled = false
    host_ip_address    = "10.30.0.1"
    prefix_length      = 24
  }
}

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}

resource "unifi_firewall_zone" "iot" {
  site_id     = data.unifi_site.main.id
  name        = "iot"
  network_ids = [unifi_network.iot.id]
}

resource "unifi_firewall_policy" "trusted_to_iot" {
  site_id              = data.unifi_site.main.id
  enabled              = true
  name                 = "trusted-to-iot"
  action               = "ALLOW"
  allow_return_traffic = true
  source_zone_id       = unifi_firewall_zone.trusted.id
  source_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.trusted.id]
    network_match_opposite = false
  }
  destination_zone_id = unifi_firewall_zone.iot.id
  destination_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.iot.id]
    network_match_opposite = false
  }
  ip_version      = "IPV4_AND_IPV6"
  logging_enabled = false
}
