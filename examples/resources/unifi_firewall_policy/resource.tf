# Manage a host-scoped service allow policy between source and destination firewall zones.
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

resource "unifi_network" "services" {
  site_id                 = data.unifi_site.main.id
  management              = "GATEWAY"
  name                    = "services"
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

resource "unifi_firewall_zone" "services" {
  site_id     = data.unifi_site.main.id
  name        = "services"
  network_ids = [unifi_network.services.id]
}

resource "unifi_firewall_policy" "trusted_to_services" {
  site_id              = data.unifi_site.main.id
  enabled              = true
  name                 = "trusted-to-services-host"
  action               = "ALLOW"
  allow_return_traffic = false
  source_zone_id       = unifi_firewall_zone.trusted.id
  source_filter = {
    type                   = "NETWORK"
    network_ids            = [unifi_network.trusted.id]
    network_match_opposite = false
  }
  destination_zone_id = unifi_firewall_zone.services.id
  destination_filter = {
    type                      = "IP_ADDRESS"
    ip_addresses              = ["10.30.0.20"]
    ip_address_match_opposite = false
    port_filter = {
      type           = "PORTS"
      match_opposite = false
      ports          = ["53", "80", "443"]
    }
  }
  ip_version = "IPV4"
  protocol_filter = {
    type        = "PRESET"
    preset_name = "TCP_UDP"
  }
  logging_enabled = false
}
