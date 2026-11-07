data "unifi_site" "default" {
  name = "Default"
}

data "unifi_network" "default" {
  site_id = data.unifi_site.default.id
  name    = "Default"
}

# Keep a small switch in the management network; its ports serve other networks.
resource "unifi_device_management_network" "switch" {
  site_id    = data.unifi_site.default.id
  device_mac = "0c:ea:14:00:00:01"
  network_id = data.unifi_network.default.id
}
