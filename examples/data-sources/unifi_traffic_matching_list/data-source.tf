# Look up an existing traffic matching list by name.
data "unifi_site" "main" {
  name = "Default"
}

data "unifi_traffic_matching_list" "web_ports" {
  site_id = data.unifi_site.main.id
  name    = "web-ports"
}
