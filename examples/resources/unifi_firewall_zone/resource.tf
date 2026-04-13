# Manage a custom firewall zone bound to one or more networks.
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

resource "unifi_firewall_zone" "trusted" {
  site_id     = data.unifi_site.main.id
  name        = "trusted"
  network_ids = [unifi_network.trusted.id]
}
