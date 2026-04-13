# Look up an existing network within a site.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_network" "trusted" {
  site_id = data.unifi_site.main.id
  name    = "trusted"
}
