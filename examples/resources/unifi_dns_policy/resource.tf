# Manage a DNS policy such as an A record override.
data "unifi_site" "main" {
  name = "Default"
}

resource "unifi_dns_policy" "printer" {
  site_id      = data.unifi_site.main.id
  type         = "A_RECORD"
  enabled      = true
  domain       = "printer.iot.internal"
  ipv4_address = "10.30.0.50"
  ttl_seconds  = 300
}
