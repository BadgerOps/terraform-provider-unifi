# Look up a LAG directly by identifier.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_lag" "uplink" {
  site_id = data.unifi_site.main.id
  id      = "00000000-0000-0000-0000-000000000000"
}
