# Look up an existing firewall policy by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_firewall_policy" "trusted_to_iot" {
  site_id = data.unifi_site.main.id
  name    = "trusted-to-iot"
}
