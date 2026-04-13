# Manage a gateway-managed network on a UniFi site.
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

    dhcp_configuration = {
      mode                            = "SERVER"
      start_ip_address                = "10.20.0.100"
      end_ip_address                  = "10.20.0.240"
      lease_time_seconds              = 86400
      dns_server_ip_addresses         = ["10.20.0.10", "1.1.1.1"]
      ping_conflict_detection_enabled = false
    }
  }
}
