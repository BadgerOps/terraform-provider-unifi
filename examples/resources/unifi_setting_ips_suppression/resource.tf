data "unifi_site" "default" {
  name = "Default"
}

resource "unifi_setting_ips_suppression" "default" {
  site_id = data.unifi_site.default.id

  alerts = [
    {
      # Only for traffic to one internal server.
      signature_id = 2016642
      category     = "emerging-webserver"
      type         = "track"
      tracking = [
        { mode = "ip", value = "192.0.2.20", direction = "dest" },
      ]
    },
    {
      # Everywhere.
      signature_id = 2100498
      category     = "emerging-attackresponse"
      type         = "all"
    },
  ]
}
