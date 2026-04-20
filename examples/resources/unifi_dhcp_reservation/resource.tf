# Manage a DHCP reservation for a client MAC address already known to the controller.
data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_dhcp_reservation" "printer" {
  site_id     = data.unifi_site.main.id
  mac_address = "aa:bb:cc:dd:ee:ff"
  fixed_ip    = "10.170.0.50"
  enabled     = true
}
