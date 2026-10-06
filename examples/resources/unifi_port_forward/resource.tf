data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_port_forward" "game" {
  site_id          = data.unifi_site.main.id
  name             = "Game UDP"
  protocol         = "udp"
  destination_port = "3074"
  forward_ip       = "192.0.2.90"
  forward_port     = "3074"
  source           = "198.51.100.0/24"
}
