# Look up a WAN interface by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_wan" "internet_1" {
  site_id = data.unifi_site.main.id
  name    = "Internet 1"
}
