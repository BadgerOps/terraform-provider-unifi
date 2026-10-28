data "unifi_radius_profile" "wpa3_enterprise" {
  site_id = data.unifi_site.main.id
  name    = "Corporate RADIUS"
}

resource "unifi_wifi_broadcast" "wpa3_enterprise" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "staff-wpa3-enterprise"
  enabled                                 = true
  client_isolation_enabled                = false
  hide_name                               = false
  uapsd_enabled                           = true
  multicast_to_unicast_conversion_enabled = false
  broadcasting_frequencies_ghz            = [5, 6]
  advertise_device_name                   = false
  arp_proxy_enabled                       = false
  band_steering_enabled                   = true
  bss_transition_enabled                  = true

  network = { type = "NATIVE" }

  security_configuration = {
    type = "WPA3_ENTERPRISE"
    radius_configuration = {
      profile_id = data.unifi_radius_profile.wpa3_enterprise.id
      nas_id     = { type = "DERIVED", source = "BSSID" }
    }
    coa_enabled          = true
    security_mode        = "DEFAULT"
    fast_roaming_enabled = true
  }
}
