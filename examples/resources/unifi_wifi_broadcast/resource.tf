# Manage a WiFi broadcast backed by a specific UniFi network.
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

resource "unifi_wifi_broadcast" "staff" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "staff"
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = false
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = false
  broadcasting_frequencies_ghz            = [2.4, 5]
  advertise_device_name                   = false
  arp_proxy_enabled                       = false
  band_steering_enabled                   = true
  bss_transition_enabled                  = true

  network = {
    type       = "SPECIFIC"
    network_id = unifi_network.trusted.id
  }

  security_configuration = {
    type                      = "WPA2_WPA3_PERSONAL"
    passphrase                = "replace-with-strong-passphrase"
    pmf_mode                  = "OPTIONAL"
    wpa3_fast_roaming_enabled = false

    sae_configuration = {
      anticlogging_threshold_seconds = 5
      sync_time_seconds              = 5
    }
  }
}
