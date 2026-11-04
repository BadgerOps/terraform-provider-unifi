data "unifi_site" "default" {
  name = "Default"
}

data "unifi_network" "family" {
  site_id = data.unifi_site.default.id
  name    = "Family"
}

# A printer on port 2 of a small switch: Family untagged, no tagged networks.
resource "unifi_device_port_override" "printer" {
  site_id           = data.unifi_site.default.id
  device_mac        = "0c:ea:14:00:00:01"
  port_idx          = 2
  name              = "Printer"
  native_network_id = data.unifi_network.family.id
  tagged_vlan_mgmt  = "block_all"
}
