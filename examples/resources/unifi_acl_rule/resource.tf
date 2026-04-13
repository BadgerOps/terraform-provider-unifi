# Manage an ACL rule enforced by UniFi switching.
data "unifi_site" "main" {
  name = "Default"
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

resource "unifi_acl_rule" "block_iot_dns" {
  site_id         = data.unifi_site.main.id
  type            = "IPV4"
  enabled         = true
  name            = "block-iot-dns"
  action          = "BLOCK"
  protocol_filter = ["TCP", "UDP"]

  source_ip_filter = {
    type        = "NETWORKS"
    network_ids = [unifi_network.iot.id]
  }

  destination_ip_filter = {
    type  = "PORTS"
    ports = [53]
  }
}
