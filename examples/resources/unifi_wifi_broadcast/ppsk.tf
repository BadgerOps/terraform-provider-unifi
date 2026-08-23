variable "iot_local_psk" {
  type      = string
  sensitive = true
}

variable "iot_internet_psk" {
  type      = string
  sensitive = true
}

resource "unifi_wifi_broadcast" "iot_ppsk" {
  site_id                                 = data.unifi_site.main.id
  type                                    = "STANDARD"
  name                                    = "iot-ppsk"
  enabled                                 = true
  client_isolation_enabled                = true
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
    type = "WPA2_PERSONAL"
    preshared_keys = [
      {
        passphrase = var.iot_local_psk
        network    = { type = "SPECIFIC", network_id = unifi_network.trusted.id }
      },
      {
        passphrase = var.iot_internet_psk
        network    = { type = "NATIVE" }
      }
    ]
  }
}
