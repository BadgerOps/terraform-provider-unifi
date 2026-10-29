# Manage the IPS alert suppressions of a site.
data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_setting_ips_suppression" "site" {
  site_id = data.unifi_site.main.id

  alerts = [
    {
      # Suppress only for traffic to one internal server.
      signature_id = 2016642
      category     = "emerging-webserver"
      type         = "track"
      tracking = [
        { mode = "ip", value = "10.20.0.50", direction = "dest" },
      ]
    },
    {
      # Suppress for traffic from the whole IoT network.
      signature_id = 2100498
      category     = "emerging-scan"
      type         = "track"
      tracking = [
        { mode = "subnet", value = "10.30.0.0/24", direction = "src" },
      ]
    },
    {
      # Suppress everywhere.
      signature_id = 2013504
      category     = "emerging-attackresponse"
      type         = "all"
    },
  ]
}
