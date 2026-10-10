data "unifi_radius_profile" "mixed_enterprise" {
  site_id = data.unifi_site.main.id
  name    = "Corporate RADIUS"
}

resource "unifi_wifi_broadcast" "mixed_enterprise" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "staff-mixed-enterprise"
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
    type = "WPA2_WPA3_ENTERPRISE"
    radius_configuration = {
      profile_id = data.unifi_radius_profile.mixed_enterprise.id
      nas_id     = { type = "USER_DEFINED", value = "staff-wifi" }
    }
    coa_enabled               = true
    pmf_mode                  = "OPTIONAL"
    fast_roaming_enabled      = true
    wpa3_fast_roaming_enabled = true
  }
}
