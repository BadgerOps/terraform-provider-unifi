# Manage a reusable traffic matching list for firewall and policy references.
data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_traffic_matching_list" "web_ports" {
  site_id = data.unifi_site.main.id
  type    = "PORTS"
  name    = "web-ports"
  ports   = ["80", "443", "8443-8444"]
}
