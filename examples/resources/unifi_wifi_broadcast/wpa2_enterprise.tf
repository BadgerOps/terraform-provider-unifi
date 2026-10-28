data "unifi_radius_profile" "wpa2_enterprise" {
  site_id = data.unifi_site.main.id
  name    = "Corporate RADIUS"
}

resource "unifi_wifi_broadcast" "wpa2_enterprise" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "staff-wpa2-enterprise"
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

  network = { type = "NATIVE" }

  security_configuration = {
    type = "WPA2_ENTERPRISE"
    radius_configuration = {
      profile_id = data.unifi_radius_profile.wpa2_enterprise.id
      nas_id     = { type = "DERIVED", source = "DEVICE_NAME" }
    }
    coa_enabled          = true
    fast_roaming_enabled = true
    pmf_mode             = "OPTIONAL"
  }
}
