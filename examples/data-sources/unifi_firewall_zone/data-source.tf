# Look up an existing firewall zone by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_firewall_zone" "trusted" {
  site_id = data.unifi_site.main.id
  name    = "trusted"
}
