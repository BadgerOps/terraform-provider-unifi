data "unifi_site" "main" {
  name = "Default"
}

data "unifi_port_forward" "game" {
  site_id = data.unifi_site.main.id
  name    = "Game UDP"
  # Alternatively, set id to the legacy rule _id and omit name.
}
