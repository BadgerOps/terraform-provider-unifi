# Look up an existing WiFi broadcast by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_wifi_broadcast" "staff" {
  site_id = data.unifi_site.main.id
  name    = "staff"
}
